# DSPy Design Patterns

Common reusable patterns for building LM pipelines.

## 1. Simple Classify

**Use case**: Text classification, sentiment analysis, topic labeling

```python
class ClassifyModule(dspy.Module):
    def __init__(self):
        self.classify = dspy.ChainOfThought("input -> classification")
    
    def forward(self, input_text):
        result = self.classify(input=input_text)
        return dspy.Prediction(label=result.classification)
```

**Training**:
```python
metric = lambda gold, pred: int(pred.label.lower() == gold.label.lower())
examples = [
    dspy.Example(input="This movie is great!", label="positive"),
    ...
]
```

---

## 2. Generate & Validate

**Use case**: Generation with automatic validation/filtering

```python
class GenerateAndValidate(dspy.Module):
    def __init__(self):
        self.generate = dspy.ChainOfThought("prompt -> output")
        self.validate = dspy.ChainOfThought("output -> is_valid")
    
    def forward(self, prompt):
        output = self.generate(prompt=prompt).output
        valid = self.validate(output=output).is_valid
        
        if "yes" in valid.lower():
            return dspy.Prediction(output=output, valid=True)
        else:
            return dspy.Prediction(output=output, valid=False)
```

---

## 3. Multi-Hop Reasoning

**Use case**: Complex questions requiring multiple steps (HotpotQA, multi-document QA)

```python
class MultiHop(dspy.Module):
    def __init__(self, num_hops=3):
        self.retrieve = dspy.Retrieve(k=3)
        self.decompose = dspy.ChainOfThought("question -> subquestions")
        self.answer = dspy.ChainOfThought("question, context -> answer")
        self.num_hops = num_hops
    
    def forward(self, question):
        # Decompose into sub-questions
        subqs_response = self.decompose(question=question)
        subquestions = subqs_response.subquestions
        
        # Retrieve for each sub-question
        all_context = []
        for subq in subquestions.split(";"):
            context = self.retrieve(subq.strip()).passages
            all_context.extend(context)
        
        # Answer with accumulated context
        result = self.answer(
            question=question,
            context="\n---\n".join(all_context[:10])
        )
        
        return dspy.Prediction(
            answer=result.answer,
            subquestions=subquestions
        )
```

---

## 4. RAG (Retrieval-Augmented Generation)

**Use case**: QA over documents, knowledge base queries

```python
class RAG(dspy.Module):
    def __init__(self, num_passages=3):
        self.retrieve = dspy.Retrieve(k=num_passages)
        self.generate = dspy.ChainOfThought(
            "context, question -> answer"
        )
    
    def forward(self, question):
        context = self.retrieve(question).passages
        prediction = self.generate(
            context="\n---\n".join(context),
            question=question
        )
        return dspy.Prediction(
            context=context,
            answer=prediction.answer
        )
```

**Metric**:
```python
def rag_metric(gold, pred):
    # F1 between gold answer and predicted answer
    gold_tokens = set(gold.answer.lower().split())
    pred_tokens = set(pred.answer.lower().split())
    
    if not pred_tokens or not gold_tokens:
        return 0
    
    precision = len(gold_tokens & pred_tokens) / len(pred_tokens)
    recall = len(gold_tokens & pred_tokens) / len(gold_tokens)
    
    if precision + recall == 0:
        return 0
    
    f1 = 2 * (precision * recall) / (precision + recall)
    return f1
```

---

## 5. Routing / Expert Selection

**Use case**: Route inputs to specialized modules (math vs english, coding vs writing)

```python
class RoutingPipeline(dspy.Module):
    def __init__(self):
        self.router = dspy.ChainOfThought("input -> category")
        self.math_solver = dspy.ChainOfThought("problem -> solution")
        self.essay_writer = dspy.ChainOfThought("prompt -> essay")
        self.code_generator = dspy.ChainOfThought("requirement -> code")
    
    def forward(self, input_text):
        # Route based on category
        category = self.router(input=input_text).category.lower()
        
        if "math" in category or "calculate" in category:
            result = self.math_solver(problem=input_text).solution
            route = "math"
        elif "code" in category or "program" in category:
            result = self.code_generator(requirement=input_text).code
            route = "coding"
        else:
            result = self.essay_writer(prompt=input_text).essay
            route = "writing"
        
        return dspy.Prediction(
            output=result,
            route=route
        )
```

---

## 6. Self-Critique & Refinement

**Use case**: Improve output quality through iteration

```python
class SelfCritique(dspy.Module):
    def __init__(self):
        self.generate = dspy.ChainOfThought("question -> answer")
        self.critique = dspy.ChainOfThought(
            "question, answer -> critique"
        )
        self.refine = dspy.ChainOfThought(
            "question, answer, critique -> refined_answer"
        )
    
    def forward(self, question):
        # Generate initial answer
        answer = self.generate(question=question).answer
        
        # Critique it
        critique = self.critique(
            question=question,
            answer=answer
        ).critique
        
        # Refine based on critique
        if "needs improvement" in critique.lower():
            refined = self.refine(
                question=question,
                answer=answer,
                critique=critique
            ).refined_answer
            return dspy.Prediction(
                answer=refined,
                refined=True
            )
        else:
            return dspy.Prediction(
                answer=answer,
                refined=False
            )
```

---

## 7. Multi-Agent Ensemble

**Use case**: Multiple perspectives on a question (consensus, voting)

```python
class EnsemblePipeline(dspy.Module):
    def __init__(self, num_agents=3):
        self.agents = [
            dspy.ChainOfThought("question -> answer")
            for _ in range(num_agents)
        ]
        self.synthesize = dspy.ChainOfThought(
            "question, answers -> final_answer"
        )
    
    def forward(self, question):
        # Get answers from multiple agents
        answers = []
        for agent in self.agents:
            result = agent(question=question).answer
            answers.append(result)
        
        # Synthesize into final answer
        synthesis = self.synthesize(
            question=question,
            answers=" | ".join(answers)
        ).final_answer
        
        return dspy.Prediction(
            answers=answers,
            final_answer=synthesis
        )
```

---

## 8. Streaming / Chunked Processing

**Use case**: Long documents, batch processing

```python
class StreamingRAG(dspy.Module):
    def __init__(self, chunk_size=3):
        self.retrieve = dspy.Retrieve(k=3)
        self.process_chunk = dspy.ChainOfThought(
            "question, chunk -> summary"
        )
        self.synthesize = dspy.ChainOfThought(
            "question, summaries -> answer"
        )
    
    def forward(self, question, documents):
        # Retrieve relevant documents
        context = self.retrieve(question).passages
        
        # Process in chunks
        summaries = []
        for i in range(0, len(context), 3):
            chunk = "\n".join(context[i:i+3])
            summary = self.process_chunk(
                question=question,
                chunk=chunk
            ).summary
            summaries.append(summary)
        
        # Final synthesis
        answer = self.synthesize(
            question=question,
            summaries="\n---\n".join(summaries)
        ).answer
        
        return dspy.Prediction(answer=answer)
```

---

## 9. Few-Shot In-Context Learning

**Use case**: Demonstrate behavior via examples

```python
class FewShotDemo(dspy.Signature):
    """Complex task with examples."""
    # Examples will be added by BootstrapFewShot
    question: str = dspy.InputField()
    answer: str = dspy.OutputField()

class FewShotModule(dspy.Module):
    def __init__(self):
        self.task = dspy.ChainOfThought(FewShotDemo)
    
    def forward(self, question):
        # ChainOfThought + demos = high-quality outputs
        return self.task(question=question)
```

**Teleprompter**:
```python
teleprompter = BootstrapFewShot(
    metric_fn=my_metric,
    max_bootstrapped_demos=6,  # Learn 6 examples
    num_candidate_programs=32,
    num_threads=16
)
```

---

## 10. Caching & Memoization

**Use case**: Reduce API calls on repeated queries

```python
class CachedRAG(dspy.Module):
    def __init__(self):
        self.retrieve = dspy.Retrieve(k=3)
        self.generate = dspy.ChainOfThought("context, question -> answer")
        self.cache = {}
    
    def forward(self, question):
        # Check cache
        if question in self.cache:
            return self.cache[question]
        
        # Compute if not cached
        context = self.retrieve(question).passages
        result = self.generate(
            context="\n---\n".join(context),
            question=question
        )
        
        prediction = dspy.Prediction(answer=result.answer)
        self.cache[question] = prediction
        return prediction
```

---

## Choosing a Pattern

| Goal | Pattern | Teleprompter |
|------|---------|--------------|
| Simple classification | Classify | BootstrapFewShot |
| QA over docs | RAG | COPRO |
| Complex reasoning | Multi-Hop | BayesianSignatureOptimizer |
| Route to experts | Routing | SignatureOptimizer |
| Improve quality | Self-Critique | BootstrapFewShot |
| Multiple perspectives | Ensemble | BootstrapFewShot |
| Long documents | Streaming | COPRO |

---

**More patterns?** Check the [DSPy docs](https://dspy.ai) or the [examples](https://github.com/stanfordnlp/dspy/tree/main/examples).
