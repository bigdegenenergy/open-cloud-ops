# DSPy Integration Guide

This toolkit now includes comprehensive support for **DSPy** — Stanford's framework for programming language models.

## What is DSPy?

DSPy enables you to **program** language models instead of just prompting them. Key features:

- **Signatures**: Declarative input/output specifications
- **Modules**: Composable building blocks for LM pipelines
- **Teleprompters**: Auto-optimize prompts and examples
- **Assertions**: Quality gates within pipelines
- **Adapters**: Support for OpenAI, Anthropic, HuggingFace, etc.

## Quick Start

### 1. Check Available Commands

All DSPy commands start with `/dspy-`:

```
/dspy-scaffold    — Generate a new DSPy project
/dspy-optimize    — Run teleprompter optimization
/dspy-evaluate    — Design evaluation metrics
/dspy-trace       — Debug pipeline execution
/dspy-compile     — Deploy to production
```

### 2. Create a DSPy Project

In Claude Code, run:
```
/dspy-scaffold
```

Answer the interview questions to generate a complete project with:
- Pipeline module (`dspy_pipeline.py`)
- Evaluator function (`evaluator.py`)
- Optimizer script (`optimizer.py`)
- Sample data and tests

### 3. Implement Your Pipeline

Edit `dspy_pipeline.py`:

```python
import dspy

class MySignature(dspy.Signature):
    """Your task description."""
    input_field: str = dspy.InputField()
    output_field: str = dspy.OutputField()

class MyPipeline(dspy.Module):
    def forward(self, input_field):
        # Your logic here
        return dspy.Prediction(output_field="result")
```

### 4. Optimize with Teleprompter

Run:
```bash
python optimizer.py
```

This learns in-context examples automatically and improves your pipeline.

### 5. Deploy

Use `/dspy-compile` to create a production-ready package.

## Agents & Roles

### DSPy Architect
**Expert**: Building optimized LM pipelines

Use when:
- Designing signatures for complex tasks
- Choosing optimization strategies
- Architecting multi-module pipelines
- Debugging optimization issues

## Core Commands

### `/dspy-scaffold`
Generates a complete DSPy project structure:
- Task signature definition
- Pipeline module
- Evaluation metrics
- Optimizer configuration
- Sample data

**When to use**: Starting a new DSPy project

**Output**: Complete, runnable project

### `/dspy-optimize`
Runs teleprompter optimization to improve pipeline quality:
- Learns in-context examples automatically
- Optimizes prompt wording
- Compares candidate programs
- Measures improvement on dev set

**When to use**: After initial implementation, want to boost quality

**Teleprompters available**:
- **BootstrapFewShot** (fast) — Learn examples from training data
- **COPRO** (thorough) — Co-optimize prompts and examples
- **SignatureOptimizer** (flexible) — Greedy prompt optimization
- **BayesianSignatureOptimizer** (advanced) — Multi-dimensional search

### `/dspy-evaluate`
Design and run evaluation metrics:
- Define quality functions
- Aggregate scores
- Run quality gates
- Generate reports

**Metrics included**:
- Exact Match (EM)
- F1 Score
- ROUGE
- Semantic Similarity
- Custom domain metrics

### `/dspy-trace`
Debug pipeline execution:
- Inspect LM prompts and outputs
- Trace module calls
- Profile latency and cost
- Identify bottlenecks

**Use when**:
- Pipeline producing unexpected outputs
- Optimization not improving
- Need to understand LM behavior
- Reducing costs

### `/dspy-compile`
Prepare pipeline for production:
- Save optimized state
- Extract final prompts
- Generate deployment bundle
- Create API wrappers
- Build Docker containers

**Output**: Production-ready packages

## Patterns & Templates

### RAG (Retrieval-Augmented Generation)
```python
# tools/dspy-templates/rag-pipeline.py
from rag_pipeline import RAGPipeline

pipeline = RAGPipeline(num_passages=3)
answer = pipeline.forward(question="Your question")
```

### Multi-Hop QA
```python
# tools/dspy-templates/multi-hop-qa.py
from multi_hop_qa import MultiHopQA

pipeline = MultiHopQA(num_hops=3)
result = pipeline.forward(question="Complex question")
```

See `docs/DSPY-PATTERNS.md` for more patterns:
- Classify
- Generate & Validate
- Routing
- Self-Critique
- Ensemble
- Streaming

## Workflow Example

### Build a Q&A System Over Documents

**Step 1: Scaffold**
```bash
/dspy-scaffold

# Answer questions:
# Task: Question answering over documents
# Inputs: question, documents
# Outputs: answer
# Metric: F1 Score
```

**Step 2: Implement**
```python
# dspy_pipeline.py
class QA(dspy.Module):
    def __init__(self):
        self.retrieve = dspy.Retrieve(k=3)
        self.answer = dspy.ChainOfThought("context, question -> answer")
    
    def forward(self, question):
        context = self.retrieve(question).passages
        return self.answer(context=context, question=question)
```

**Step 3: Optimize**
```bash
/dspy-optimize

# Runs BootstrapFewShot automatically
# Output: 45% → 62% F1 improvement
```

**Step 4: Deploy**
```bash
/dspy-compile

# Creates production package with:
# - Saved pipeline
# - Extracted prompts
# - API server
# - Docker image
```

## GitHub Actions

### Automatic Quality Gates

PR trigger: `.github/workflows/dspy-test-optimize.yml`

When you push DSPy code:
1. Runs unit tests
2. Optimizes pipeline automatically
3. Compares against baseline
4. Comments with improvement %
5. Fails if quality drops >5%

## Resources

- **Quick Start**: `docs/DSPY-QUICKSTART.md`
- **Design Patterns**: `docs/DSPY-PATTERNS.md`
- **Templates**: `tools/dspy-templates/`
- **Official Docs**: https://dspy.ai
- **GitHub**: https://github.com/stanfordnlp/dspy
- **Discord**: https://discord.gg/XCGy2WDCQB

## Common Tasks

### Train & Optimize Pipeline
```bash
# From scaffold directory
python optimizer.py

# Output: optimized_pipeline.pkl
```

### Test Locally
```bash
pytest tests/ -v
python dspy_pipeline.py
```

### Deploy to Production
```bash
/dspy-compile

# Outputs deployment bundle:
# - pipeline.pkl
# - requirements.txt
# - api.py (FastAPI)
# - Dockerfile
# - config.json
```

### Monitor Quality Over Time
```bash
# Save metrics before & after optimization
python evaluate.py --save-metrics

# Track improvements in dashboard
```

## Troubleshooting

**Q: Optimization not improving quality**
- A: Check evaluator metric matches your task
- A: Increase training data size
- A: Try different teleprompter (COPRO instead of Bootstrap)

**Q: Pipeline too slow**
- A: Reduce number of LM calls
- A: Cache retrievals
- A: Use faster model (GPT-3.5 vs GPT-4)

**Q: Prompts not improving**
- A: Increase `max_bootstrapped_demos`
- A: Provide more training examples
- A: Adjust `num_candidate_programs`

**Q: Memory issues**
- A: Reduce batch size
- A: Process documents in chunks
- A: Use streaming teleprompter

## Contributing Patterns

Have a useful DSPy pattern? Add it to `tools/dspy-templates/`:

1. Create `your-pattern.py` with working example
2. Include docstring + inline comments
3. Add usage example at bottom
4. Reference in `DSPY-PATTERNS.md`

## Next Steps

1. ✅ Understand DSPy concepts (`DSPY-QUICKSTART.md`)
2. ✅ Explore patterns (`DSPY-PATTERNS.md`)
3. ✅ Build your first pipeline (`/dspy-scaffold`)
4. ✅ Optimize (`/dspy-optimize`)
5. ✅ Deploy (`/dspy-compile`)
6. ✅ Monitor & iterate

---

**Questions?** Check the DSPy Discord or open an issue on GitHub.

**Happy programming with DSPy! 🚀**
