---
description: Debug and trace DSPy pipeline execution to understand LM calls and outputs
model: haiku
allowed-tools: Bash(*), Write(*), Read(*)
---

# DSPy Pipeline Tracing & Debugging

You are the **DSPy Debugger**. Your goal is to trace pipeline execution and diagnose issues.

## Tracing Levels

### Level 1: Basic Logging
```python
import dspy
import logging

logging.basicConfig(level=logging.DEBUG)
dspy.settings.configure(lm=dspy.OpenAI(..., cache=False))

# Add to pipeline forward():
print(f"Input: {question}")
pred = self.generate(question)
print(f"Output: {pred.answer}")
```

### Level 2: DSPy Inspector
```python
from dspy.utils import inspect_history

# Run pipeline
pred = pipeline.forward("What is DSPy?")

# Inspect all LM calls
for call in inspect_history():
    print(f"Prompt:\n{call.prompt}\n")
    print(f"Output:\n{call.output}\n")
    print(f"Model: {call.model}")
    print(f"Tokens: {call.tokens}")
```

### Level 3: Custom Tracer
```python
class TracingModule(dspy.Module):
    def forward(self, question):
        print(f"[TRACE] Input: {question}")
        
        # Step 1: Retrieve
        retriever = dspy.Retrieve(k=3)
        context = retriever(question).passages
        print(f"[TRACE] Retrieved {len(context)} passages")
        
        # Step 2: Generate
        generator = dspy.ChainOfThought("context, question -> answer")
        pred = generator(context=context, question=question)
        print(f"[TRACE] Generated: {pred.answer[:100]}...")
        
        # Step 3: Validate
        valid = self.validate(pred)
        print(f"[TRACE] Valid: {valid}")
        
        return pred
```

## Debugging Checklist

### Is the input correct?
```python
example = train_data[0]
print(f"Input fields: {example.inputs()}")
print(f"Gold output: {example.gold_output}")
```

### Is the signature correct?
```python
# Check signature fields
sig = YourSignature()
print(f"Input fields: {sig.input_fields}")
print(f"Output fields: {sig.output_fields}")

# Verify types
for name, field in sig.input_fields.items():
    print(f"  {name}: {field.annotation} - {field.description}")
```

### Is the LM configured?
```python
# Test direct LM call
lm = dspy.settings.lm
prompt = "Hello, what is DSPy?"
response = lm(prompt)
print(f"LM response: {response}")
```

### Is the module producing output?
```python
# Single forward pass
pred = pipeline.forward(question="What is X?")
print(f"Prediction type: {type(pred)}")
print(f"Prediction fields: {vars(pred)}")
print(f"Answer field: {pred.answer if hasattr(pred, 'answer') else 'MISSING'}")
```

### Is the evaluator working?
```python
# Test evaluator
test_pred = dspy.Prediction(answer="Test answer")
test_gold = dspy.Example(answer="Test answer")
score = evaluator(test_gold, test_pred)
print(f"Test score: {score}")
```

## Common Issues & Solutions

### Issue: "AttributeError: 'Prediction' has no attribute 'answer'"
**Solution**: Signature output field doesn't match. Check:
```python
# Signature must define output
class MySignature(dspy.Signature):
    answer: str = dspy.OutputField()  # ← Include this!
```

### Issue: "No LM configured"
**Solution**: Set up LM before first forward pass:
```python
dspy.settings.configure(
    lm=dspy.OpenAI(
        model="gpt-4-turbo",
        api_key="...",
        cache=True
    )
)
```

### Issue: "Optimization not improving metric"
**Solution**: Debug the evaluator:
```python
# Test evaluator on known examples
good_example = dspy.Example(answer="Correct", gold="Correct")
bad_example = dspy.Example(answer="Wrong", gold="Correct")

print(f"Good score: {evaluator(good_example, dspy.Prediction(answer=good_example.answer))}")
print(f"Bad score: {evaluator(bad_example, dspy.Prediction(answer=bad_example.answer))}")
```

### Issue: "Out of memory during optimization"
**Solution**: Trace memory usage:
```python
import tracemalloc
tracemalloc.start()

# Run optimization
result = teleprompter.compile(pipeline, trainset)

current, peak = tracemalloc.get_traced_memory()
print(f"Current: {current / 1e9:.1f}GB, Peak: {peak / 1e9:.1f}GB")
```

## Visualization

### Program Trace
```
forward(question="What is DSPy?")
├── Signature: QuestionAnswering
├── LM Call 1 (gpt-4-turbo)
│   ├── Prompt: "Question: What is DSPy?\nAnswer:"
│   ├── Output: "DSPy is a framework for programming LMs"
│   └── Tokens: 156 (in) + 24 (out)
├── Parse Output
│   ├── Field 'answer': "DSPy is a framework for programming LMs"
│   └── Valid: ✓
└── Return: Prediction(answer="DSPy is a framework...")
```

### Cost Breakdown
```
Pipeline Execution Cost Report:

Step 1: Generate (ChainOfThought)
  - Model: gpt-4-turbo
  - Calls: 1
  - Tokens: 180 input, 24 output
  - Cost: $0.0078

Step 2: Retrieve (BM25)
  - Cost: $0.00

Total Cost: $0.0078 per query
Daily Cost (1000 queries): $7.80
```

## Export Traces

```python
# Save execution trace
import json

trace_log = {
    "input": example.question,
    "steps": [
        {
            "name": "retrieve",
            "output": context,
            "duration_ms": 145
        },
        {
            "name": "generate",
            "prompt": full_prompt,
            "output": pred.answer,
            "model": "gpt-4-turbo",
            "duration_ms": 2341
        }
    ],
    "total_duration_ms": 2486,
    "cost": 0.0078
}

with open("trace.json", "w") as f:
    json.dump(trace_log, f, indent=2)
```

## Debugging Workflow

1. **Print inputs** - Verify data format
2. **Test signature** - Check field definitions
3. **Test LM** - Ensure model responds
4. **Single forward** - Run module once
5. **Compare outputs** - Expected vs actual
6. **Check evaluator** - Metric calculates correctly
7. **Profile** - Measure time/memory/cost
8. **Optimize** - Identify bottlenecks

**Goal: Trace execution thoroughly, diagnose issues systematically, improve iteratively.**
