# DSPy Quick Start Guide

DSPy is a framework for **programming** language models, not just prompting them. This guide gets you up and running in 10 minutes.

## Installation

```bash
pip install dspy-ai
```

## 1. Define Your Task (Signature)

A signature defines the input/output spec for your task:

```python
import dspy

class GenerateAnswer(dspy.Signature):
    """Answer questions based on provided context."""
    context: str = dspy.InputField(desc="May contain relevant information")
    question: str = dspy.InputField()
    answer: str = dspy.OutputField(desc="Often between 1-5 sentences")
```

## 2. Create a Module

A module is a composable building block:

```python
class RagPipeline(dspy.Module):
    def __init__(self, num_passages):
        super().__init__()
        self.retrieve = dspy.Retrieve(k=num_passages)
        self.generate_answer = dspy.ChainOfThought(GenerateAnswer)

    def forward(self, question):
        context = self.retrieve(question).passages
        prediction = self.generate_answer(context="\n---\n".join(context), question=question)
        return dspy.Prediction(context=context, answer=prediction.answer)
```

## 3. Configure LM

```python
dspy.settings.configure(
    lm=dspy.OpenAI(
        model="gpt-4-turbo",
        api_key="your-api-key",
        cache=True  # Cache for faster iteration
    )
)
```

## 4. Test Basic Inference

```python
# Create pipeline
pipeline = RagPipeline(num_passages=3)

# Test
result = pipeline.forward(question="What is DSPy?")
print(f"Answer: {result.answer}")
```

## 5. Define Evaluation Metric

```python
def metric(gold, pred, trace=None):
    """Simple exact match metric."""
    answer = pred.answer.lower()
    gold_answer = gold.answer.lower()
    return int(answer == gold_answer)
```

## 6. Optimize with Teleprompter

```python
from dspy.teleprompt import BootstrapFewShot

# Load your training data
train_examples = [
    dspy.Example(
        context="DSPy is...",
        question="What is DSPy?",
        answer="DSPy is a framework for programming LMs"
    ),
    # ... more examples
]

# Create teleprompter
teleprompter = BootstrapFewShot(
    metric_fn=metric,
    max_bootstrapped_demos=4,
    num_candidate_programs=16
)

# Optimize
optimized_pipeline = teleprompter.compile(
    pipeline,
    trainset=train_examples,
    valset=train_examples[:50]
)
```

## 7. Evaluate Results

```python
# Test on dev set
dev_examples = [...]  # Your test data

scores = []
for example in dev_examples:
    pred = optimized_pipeline.forward(question=example.question)
    score = metric(example, pred)
    scores.append(score)

avg_score = sum(scores) / len(scores)
print(f"Dev Score: {avg_score:.2%}")
```

## 8. Save for Production

```python
import json

# Save checkpoint using DSPy's safe serialization
state = optimized_pipeline.dump_state()
with open('optimized_pipeline.json', 'w') as f:
    json.dump(state, f, indent=2)

# Load and use
# Reconstruct pipeline from saved state at runtime
# (Requires re-instantiation with API key for security)
with open('optimized_pipeline.json', 'r') as f:
    state = json.load(f)
    # Restore state to fresh pipeline instance
    # See framework docs for load_state() or equivalent
```

**Security Note**: Avoid `pickle.dump()` for serialization. Use JSON with DSPy's `dump_state()` to prevent arbitrary code execution vulnerabilities.

---

## Key Concepts

### Signature
Declares what a module does (input/output spec):
```python
class MyTask(dspy.Signature):
    input_field: str = dspy.InputField(desc="...")
    output_field: str = dspy.OutputField(desc="...")
```

### Module
Implements a reusable component:
```python
class MyModule(dspy.Module):
    def forward(self, input_field):
        # Your logic here
        return dspy.Prediction(output_field=...)
```

### ChainOfThought
Adds reasoning steps before output:
```python
generate = dspy.ChainOfThought(MySignature)
result = generate(input_field="...")
# Produces: "Let me think... [reasoning] [output]"
```

### Prediction
Result object with named fields:
```python
pred = dspy.Prediction(answer="DSPy is...", confidence=0.95)
pred.answer  # "DSPy is..."
```

### Teleprompter
Auto-optimizer for your pipeline:
- **BootstrapFewShot**: Learn examples from training data
- **COPRO**: Co-optimize prompts and examples
- **SignatureOptimizer**: Simple prompt optimization

---

## Common Patterns

### RAG (Retrieval-Augmented Generation)
```python
class RAG(dspy.Module):
    def __init__(self):
        self.retrieve = dspy.Retrieve(k=3)
        self.generate = dspy.ChainOfThought("context, question -> answer")
    
    def forward(self, question):
        context = self.retrieve(question).passages
        answer = self.generate(context=context, question=question).answer
        return dspy.Prediction(answer=answer)
```

### Multi-Hop Reasoning
```python
class MultiHop(dspy.Module):
    def __init__(self):
        self.retrieve = dspy.Retrieve(k=3)
        self.decompose = dspy.ChainOfThought("question -> subquestions")
        self.answer = dspy.ChainOfThought("subquestions, context -> answer")
    
    def forward(self, question):
        subqs = self.decompose(question=question).subquestions
        context = self.retrieve(subqs).passages
        answer = self.answer(subquestions=subqs, context=context).answer
        return dspy.Prediction(answer=answer)
```

### Routing
```python
class Router(dspy.Module):
    def __init__(self):
        self.classify = dspy.ChainOfThought("question -> category")
        self.qa = dspy.ChainOfThought("question -> answer")
        self.code = dspy.ChainOfThought("question -> code")
    
    def forward(self, question):
        category = self.classify(question=question).category
        
        if "code" in category.lower():
            result = self.code(question=question).code
        else:
            result = self.qa(question=question).answer
        
        return dspy.Prediction(result=result)
```

---

## Next Steps

1. **Define your task** — Create signature for your problem
2. **Build pipeline** — Combine modules for your approach
3. **Optimize** — Run teleprompter on training data
4. **Evaluate** — Measure quality on test set
5. **Deploy** — Save pipeline and integrate to production

## Resources

- **Docs**: [dspy.ai](https://dspy.ai)
- **GitHub**: [stanfordnlp/dspy](https://github.com/stanfordnlp/dspy)
- **Discord**: [DSPy Community](https://discord.gg/XCGy2WDCQB)
- **Papers**: [DSPy Research](https://github.com/stanfordnlp/dspy#citation)

---

**Happy programming! Questions? Start a discussion or check examples in the DSPy repo.**
