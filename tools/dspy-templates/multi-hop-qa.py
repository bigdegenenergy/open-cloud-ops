"""
Multi-Hop Question Answering Pipeline Template
Use for complex questions requiring multiple retrieval steps (HotpotQA, etc.)
"""

import dspy


class DecomposeSignature(dspy.Signature):
    """Break down a complex question into simpler sub-questions."""
    question: str = dspy.InputField(desc="Complex question")
    subquestions: str = dspy.OutputField(
        desc="List of simpler questions (separated by newline)"
    )


class AnswerSignature(dspy.Signature):
    """Answer a question given relevant context."""
    context: str = dspy.InputField(desc="Retrieved context passages")
    question: str = dspy.InputField(desc="The question to answer")
    answer: str = dspy.OutputField(desc="Clear, factual answer")


class MultiHopQA(dspy.Module):
    """
    Multi-hop question answering pipeline.
    
    Decomposes complex questions, retrieves context for each sub-question,
    then synthesizes final answer.
    """
    
    def __init__(self, num_passages: int = 3, num_hops: int = 3):
        super().__init__()
        self.decompose = dspy.ChainOfThought(DecomposeSignature)
        self.retrieve = dspy.Retrieve(k=num_passages)
        self.answer = dspy.ChainOfThought(AnswerSignature)
        self.num_passages = num_passages
        self.num_hops = num_hops
    
    def forward(self, question: str) -> dspy.Prediction:
        """
        Multi-hop forward pass:
        1. Decompose question into sub-questions
        2. Retrieve context for each sub-question
        3. Synthesize final answer
        
        Args:
            question: Complex question requiring multiple hops
        
        Returns:
            Prediction with answer, sub-questions, and context
        """
        # Step 1: Decompose into sub-questions
        decomp = self.decompose(question=question)
        subquestions = decomp.subquestions
        
        # Parse sub-questions with basic LLM formatting handling
        import re
        subq_list = []
        for line in subquestions.split("\n"):
            # Remove common list prefixes: "1.", "1)", "- ", "* ", etc.
            # Note: This regex is fragile for conversational responses (e.g., "Here are the steps: 1. ...")
            # For production, consider using DSPy TypedPredictor with List[str] output
            # to let the framework handle serialization/deserialization
            cleaned = re.sub(r'^[\d]+[\.\)]?\s*', '', line.strip())
            cleaned = re.sub(r'^[\-\*]\s*', '', cleaned)
            if cleaned:
                subq_list.append(cleaned)
        
        subq_list = subq_list[:self.num_hops]
        
        # Step 2: Retrieve context for each sub-question
        all_context = []
        context_by_hop = {}
        
        for hop, subq in enumerate(subq_list, 1):
            retrieved = self.retrieve(subq)
            passages = retrieved.passages[:self.num_passages]
            context_by_hop[hop] = passages
            all_context.extend(passages)
        
        # Step 3: Generate final answer with accumulated context
        context_str = "\n---\n".join(all_context[:10])  # Limit context size
        
        result = self.answer(
            context=context_str,
            question=question
        )
        
        return dspy.Prediction(
            question=question,
            subquestions=subq_list,
            context_by_hop=context_by_hop,
            answer=result.answer
        )


# ============================================================================
# EVALUATION METRICS
# ============================================================================

def f1_score(gold, pred, trace=None):
    """F1 score between gold and predicted answer."""
    gold_tokens = set(gold.answer.lower().split())
    pred_tokens = set(pred.answer.lower().split())
    
    if not pred_tokens or not gold_tokens:
        return 0.0
    
    overlap = gold_tokens & pred_tokens
    precision = len(overlap) / len(pred_tokens)
    recall = len(overlap) / len(gold_tokens)
    
    if precision + recall == 0:
        return 0.0
    
    return 2 * (precision * recall) / (precision + recall)


def multi_hop_metric(gold, pred, trace=None):
    """
    Multi-hop specific metric:
    - Bonus for correct subquestion decomposition
    - Primary score from answer quality
    """
    # Answer quality (primary)
    answer_score = f1_score(gold, pred, trace)
    
    # Subquestion quality (bonus)
    bonus = 0.0
    if hasattr(pred, 'subquestions') and hasattr(gold, 'subquestions'):
        if len(pred.subquestions) == len(gold.subquestions):
            bonus = 0.1  # Bonus for correct number of hops
    
    return min(1.0, answer_score + bonus)


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
    pipeline = MultiHopQA(num_passages=3, num_hops=2)
    
    # Example complex question
    question = "What companies were founded by AI researchers at Stanford?"
    
    result = pipeline.forward(question=question)
    
    print(f"Question: {question}")
    print("\nSub-questions identified:")
    for i, subq in enumerate(result.subquestions, 1):
        print(f"  [{i}] {subq}")
    
    print(f"\nAnswer: {result.answer}")
    
    # ========================================================================
    # OPTIMIZATION EXAMPLE
    # ========================================================================
    
    # Create training examples (HotpotQA format)
    train_examples = [
        dspy.Example(
            question="Who is the CEO of the company founded by Brin?",
            subquestions=["What company did Brin found?", "Who is the CEO of that company?"],
            answer="Sundar Pichai"
        ),
        # Add more examples...
    ]
    
    # Run optimization
    from dspy.teleprompt import BootstrapFewShot
    
    teleprompter = BootstrapFewShot(
        metric_fn=multi_hop_metric,
        max_bootstrapped_demos=4,
        num_candidate_programs=16,
        num_threads=8
    )
    
    print("\n" + "="*60)
    print("OPTIMIZING PIPELINE")
    print("="*60)
    
    optimized = teleprompter.compile(
        pipeline,
        trainset=train_examples,
        valset=train_examples[:5]
    )
    
    # Test optimized
    result = optimized.forward(question=question)
    print(f"Optimized answer: {result.answer}")
    
    # Save checkpoint using DSPy's safe serialization
    # Use save() method which uses JSON (not pickle)
    state = optimized.dump_state()
    import json
    with open("multihop_qa_optimized.json", "w") as f:
        json.dump(state, f, indent=2)
    
    print("\n✅ Pipeline state saved to multihop_qa_optimized.json (safe JSON, no pickle)")
