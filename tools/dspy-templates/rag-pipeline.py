"""
RAG (Retrieval-Augmented Generation) Pipeline Template
Use for QA over documents, knowledge base queries
"""

import dspy


class RAGSignature(dspy.Signature):
    """Answer questions using retrieved documents."""
    context: str = dspy.InputField(
        desc="Retrieved documents or context passages"
    )
    question: str = dspy.InputField(desc="The question to answer")
    answer: str = dspy.OutputField(
        desc="Clear, factual answer based on context"
    )


class RAGPipeline(dspy.Module):
    """
    Retrieval-Augmented Generation pipeline.
    
    Retrieves relevant documents then generates answer from context.
    """
    
    def __init__(self, num_passages: int = 3):
        super().__init__()
        self.retrieve = dspy.Retrieve(k=num_passages)
        self.generate_answer = dspy.ChainOfThought(RAGSignature)
        self.num_passages = num_passages
    
    def forward(self, question: str) -> dspy.Prediction:
        """
        Forward pass: retrieve documents, then generate answer.
        
        Args:
            question: The question to answer
        
        Returns:
            Prediction with answer and retrieved context
        """
        # Retrieve relevant passages
        context_obj = self.retrieve(question)
        passages = context_obj.passages[:self.num_passages]
        
        # Format context
        context = "\n---\n".join(passages)
        
        # Generate answer using retrieved context
        prediction = self.generate_answer(
            context=context,
            question=question
        )
        
        return dspy.Prediction(
            context=passages,
            answer=prediction.answer
        )


# ============================================================================
# EVALUATION & METRICS
# ============================================================================

def f1_metric(gold, pred, trace=None):
    """
    F1 score between gold and predicted answer.
    Token-level overlap metric.
    """
    gold_tokens = set(gold.answer.lower().split())
    pred_tokens = set(pred.answer.lower().split())
    
    if not pred_tokens or not gold_tokens:
        return 0.0
    
    # Calculate precision and recall
    overlap = gold_tokens & pred_tokens
    precision = len(overlap) / len(pred_tokens)
    recall = len(overlap) / len(gold_tokens)
    
    if precision + recall == 0:
        return 0.0
    
    # Return F1
    f1 = 2 * (precision * recall) / (precision + recall)
    return f1


def exact_match(gold, pred, trace=None):
    """Exact string match after normalization."""
    gold_norm = gold.answer.lower().strip()
    pred_norm = pred.answer.lower().strip()
    return 1.0 if gold_norm == pred_norm else 0.0


def combined_metric(gold, pred, trace=None):
    """Combine exact match (stricter) with F1 (lenient)."""
    em = exact_match(gold, pred, trace)
    if em > 0:
        return 1.0  # Perfect score for exact match
    
    f1 = f1_metric(gold, pred, trace)
    return f1 * 0.5  # Half credit for partial match


# ============================================================================
# EXAMPLE USAGE
# ============================================================================

if __name__ == "__main__":
    # Configure LM
    dspy.settings.configure(
        lm=dspy.OpenAI(
            model="gpt-4-turbo",
            # api_key="your-api-key",  # Set via environment variable
            cache=True
        )
    )
    
    # Create pipeline
    pipeline = RAGPipeline(num_passages=3)
    
    # Example question
    result = pipeline.forward(
        question="What is DSPy?"
    )
    
    print("Question: What is DSPy?")
    print("\nContext retrieved:")
    for i, passage in enumerate(result.context, 1):
        print(f"  [{i}] {passage[:100]}...")
    print(f"\nAnswer: {result.answer}")
    
    # ========================================================================
    # OPTIMIZATION EXAMPLE
    # ========================================================================
    
    # Create training examples
    train_examples = [
        dspy.Example(
            question="What is DSPy?",
            context="DSPy is a framework for programming LMs. It enables composable optimization.",
            answer="DSPy is a framework for programming language models"
        ),
        # Add more examples...
    ]
    
    # Run teleprompter optimization
    from dspy.teleprompt import BootstrapFewShot
    
    teleprompter = BootstrapFewShot(
        metric_fn=combined_metric,
        max_bootstrapped_demos=4,
        num_candidate_programs=16,
        num_threads=8
    )
    
    # Compile (optimize) pipeline
    optimized = teleprompter.compile(
        pipeline,
        trainset=train_examples,
        valset=train_examples[:5]
    )
    
    # Test optimized version
    print("\n" + "="*60)
    print("OPTIMIZED PIPELINE")
    print("="*60)
    
    result = optimized.forward(question="What is DSPy?")
    print(f"Optimized answer: {result.answer}")
    
    # Save checkpoint using DSPy's safe serialization
    # DSPy provides save/load methods that don't require pickle
    # For production, export the program state only (no secrets)
    state = optimized.dump_state()
    import json
    with open("rag_pipeline_optimized.json", "w") as f:
        json.dump(state, f, indent=2)
    
    print("\n✅ Pipeline state saved to rag_pipeline_optimized.json (safe JSON, no pickle)")
