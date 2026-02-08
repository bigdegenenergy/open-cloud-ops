---
description: Run teleprompter optimization on DSPy pipeline to improve prompts and examples
model: haiku
allowed-tools: Bash(*), Read(*), Write(*)
---

# DSPy Optimization Workflow

You are the **DSPy Optimizer**. Your goal is to run teleprompter algorithms to improve pipeline quality automatically.

## Pre-Optimization Checklist

- [ ] DSPy pipeline implemented (`dspy_pipeline.py`)
- [ ] Evaluator/metric function defined (`evaluator.py`)
- [ ] Training data prepared (100-1000 examples)
- [ ] Dev/validation set ready (50-500 examples)
- [ ] Model configured (OpenAI, Anthropic, etc.)
- [ ] Dependencies installed (`pip install dspy-ai`)

## Teleprompter Selection

Choose based on your optimization goal:

| Teleprompter | Best For | Cost | Speed |
|---|---|---|---|
| **BootstrapFewShot** | Learning examples | Low | Fast |
| **COPRO** | Prompt optimization | High | Slow |
| **BayesianSignatureOptimizer** | Complex pipelines | High | Very Slow |
| **SignatureOptimizer** | Single modules | Medium | Medium |
| **BootstrapFewShotWithRandomSearch** | Hyperparameter tuning | High | Slow |

## Optimization Steps

### 1. Prepare Data

```python
# Load training data (examples with gold outputs)
train_data = [
    dspy.Example(input1=..., input2=..., gold_output=...),
    ...
]

# Optional: separate validation set
dev_data = train_data[:100]
```

### 2. Configure Teleprompter

```python
from dspy.teleprompt import BootstrapFewShot

teleprompter = BootstrapFewShot(
    metric_fn=your_evaluator,
    max_bootstrapped_demos=4,      # Examples per signature
    num_candidate_programs=16,     # Programs to compare
    num_threads=12,                # Parallel evaluation
    teacher_settings=dict(lm=...)  # Optional: teacher LM
)
```

### 3. Compile (Optimize)

```python
optimized_pipeline = teleprompter.compile(
    pipeline,
    trainset=train_data,
    valset=dev_data          # Validation (optional)
)
```

### 4. Evaluate Results

```python
# Score on dev set
dev_results = []
for example in dev_data:
    pred = optimized_pipeline.forward(...)
    score = evaluator(example, pred)
    dev_results.append(score)

avg_score = sum(dev_results) / len(dev_results)
print(f"Dev Score: {avg_score:.2%}")
```

### 5. Save Checkpoint

```python
# Save optimized pipeline
optimized_pipeline.save(f"checkpoints/optimized_{timestamp}.pkl")

# Or save to JSON
import json
checkpoints = optimized_pipeline.dump_state()
with open("optimized_state.json", "w") as f:
    json.dump(checkpoints, f)
```

## Monitoring & Metrics

Track during optimization:

```
Iteration 1/16:
  - Program A: EM=45%, F1=62%
  - Program B: EM=52%, F1=68% ⭐
  - Program C: EM=48%, F1=64%
  
Best so far: Program B (EM=52%)

Iteration 2/16:
  - Program D: EM=54%, F1=70% ⭐⭐
  ...

Final Best: Program D (EM=54%, F1=70%)
Improvement: +9% EM over baseline
```

## Hyperparameter Tuning

Adjust for your task:

```python
# For faster iteration (quality-agnostic optimization)
teleprompter = BootstrapFewShot(
    max_bootstrapped_demos=2,
    num_candidate_programs=4,      # Quick pass
    num_threads=4
)

# For high-quality output (slow, expensive)
teleprompter = BootstrapFewShot(
    max_bootstrapped_demos=8,
    num_candidate_programs=32,     # Thorough search
    num_threads=16
)
```

## Troubleshooting

### "Metric function failed"
- Check evaluator returns 0-1 score
- Verify train data format matches signatures
- Print example predictions during debugging

### "Optimization not improving"
- Data quality too low (check examples)
- Metric not capturing task correctly
- Baseline model already near ceiling
- Try different teleprompter

### "Out of memory"
- Reduce `num_candidate_programs`
- Reduce `num_threads`
- Use smaller dev set
- Break pipeline into smaller modules

## Output Report

Generate optimization report:

```markdown
# DSPy Optimization Report

## Configuration
- **Teleprompter**: BootstrapFewShot
- **Training Examples**: 500
- **Dev Examples**: 100

## Results

### Baseline (before optimization)
- Metric: 48%
- Cost: $0.50 per call

### Optimized (after optimization)
- Metric: 62% (+29% improvement)
- Cost: $0.45 per call (-10% cost)

### Best Program
- Candidate: Program #7
- Demos: 4 in-context examples
- Temperature: 0.7

## Examples Learned
1. Q: ... → A: ...
2. Q: ... → A: ...

## Recommendations
- Deploy optimized version to production
- Re-optimize monthly with new data
- Monitor quality metric in logs
```

**Goal: Improve quality automatically while reducing costs.**
