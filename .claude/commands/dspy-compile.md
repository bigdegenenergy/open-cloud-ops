---
description: Compile optimized DSPy pipeline to production deployment format
model: haiku
allowed-tools: Write(*), Read(*), Bash(*)
---

# DSPy Pipeline Compilation

You are the **DSPy Compiler**. Your goal is to prepare DSPy pipelines for production deployment.

## What is Compilation?

**Compilation** converts an optimized DSPy pipeline into:

1. **Saved checkpoints** - Frozen pipeline state + learned examples
2. **Serialized prompts** - Final prompts without framework overhead
3. **Deployment bundles** - Self-contained packages for production
4. **API wrappers** - REST/gRPC endpoints for external callers

## Step 1: Save Optimized State

### JSON Format (Recommended - Language-agnostic)
```python
def pipeline_to_json(pipeline):
    """Convert DSPy pipeline to JSON."""
    state = pipeline.dump_state()
    
    # state contains:
    # - signatures
    # - learned examples (from BootstrapFewShot)
    # - module weights/parameters
    
    import json
    with open('checkpoints/pipeline_v1.json', 'w') as f:
        json.dump(state, f, indent=2)

# Load in another language
import json
with open('checkpoints/pipeline_v1.json', 'r') as f:
    state = json.load(f)
    # Reconstruct pipeline from state
```

## Step 2: Extract Prompts

### Get Final Prompts (No Framework Overhead)
```python
def extract_prompts(pipeline):
    """Extract learned prompts from compiled pipeline."""
    
    prompts = {}
    
    # Iterate through pipeline modules
    for name, module in pipeline.named_modules():
        if hasattr(module, 'signature'):
            sig = module.signature
            
            # Get learned examples
            if hasattr(module, 'demos'):
                prompts[name] = {
                    'signature': str(sig),
                    'demos': [
                        {
                            'input': demo.inputs(),
                            'output': demo.outputs()
                        }
                        for demo in module.demos
                    ]
                }
    
    return prompts

# Save prompts
prompts = extract_prompts(optimized_pipeline)
with open('prompts.json', 'w') as f:
    json.dump(prompts, f, indent=2)
```

### Generate Inference Prompt Template
```python
def generate_inference_prompt(signature, demos):
    """Create ready-to-use prompt for inference."""
    
    prompt = f"""Task: {signature.description}

Examples:
"""
    for i, demo in enumerate(demos, 1):
        inputs = demo['input']
        outputs = demo['output']
        
        prompt += f"\nExample {i}:\n"
        for k, v in inputs.items():
            prompt += f"{k}: {v}\n"
        for k, v in outputs.items():
            prompt += f"{k}: {v}\n"
        prompt += "---\n"
    
    prompt += """Now generate output for:
[USER INPUT HERE]

Output:"""
    
    return prompt
```

## Step 3: Create Deployment Bundle

```
deployment/
├── pipeline.pkl                    # Optimized DSPy pipeline
├── prompts.json                    # Extracted prompts
├── config.json                     # Configuration (models, API keys, etc)
├── requirements.txt                # Python dependencies
├── inference.py                    # Inference script
├── api.py                          # FastAPI wrapper
├── docker-compose.yml              # Docker deployment
├── README.md                        # Deployment guide
└── examples/
    ├── input.json                  # Sample input
    └── output.json                 # Expected output
```

### config.json
```json
{
  "pipeline": {
    "type": "dspy",
    "checkpoint": "pipeline.pkl",
    "version": "v1.0"
  },
  "lm": {
    "type": "openai",
    "model": "gpt-4-turbo",
    "temperature": 0.7,
    "max_tokens": 256,
    "timeout": 30
  },
  "retriever": {
    "type": "bm25",
    "index_path": "data/index",
    "top_k": 3
  },
  "performance": {
    "dev_score": 0.624,
    "avg_latency_ms": 2341,
    "cost_per_query": 0.0078
  }
}
```

### inference.py
```python
import pickle
import json
from pathlib import Path

class PipelineServer:
    def __init__(self, checkpoint_path, config_path):
        # Load pipeline
        with open(checkpoint_path, 'rb') as f:
            self.pipeline = pickle.load(f)['pipeline']
        
        # Load config
        with open(config_path, 'r') as f:
            self.config = json.load(f)
    
    def predict(self, question, context=None):
        """Run inference."""
        result = self.pipeline.forward(
            question=question,
            context=context
        )
        return {
            'answer': result.answer,
            'confidence': getattr(result, 'confidence', None)
        }

# Usage
server = PipelineServer('pipeline.pkl', 'config.json')
output = server.predict("What is DSPy?")
print(output)
```

### api.py (FastAPI)
```python
from fastapi import FastAPI
from pydantic import BaseModel
import uvicorn
import json

app = FastAPI()

def load_pipeline_state(state_path):
    """Load DSPy pipeline state from JSON (safe, not pickle)."""
    with open(state_path, 'r') as f:
        return json.load(f)

server = PipelineServer(load_pipeline_state('pipeline.json'), 'config.json')

class QueryRequest(BaseModel):
    question: str
    context: str = None

class QueryResponse(BaseModel):
    answer: str
    confidence: float = None

@app.post("/predict", response_model=QueryResponse)
async def predict(request: QueryRequest):
    result = server.predict(request.question, request.context)
    return QueryResponse(**result)

@app.get("/health")
async def health():
    return {"status": "ok"}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
```

## Step 4: Version Management

```
checkpoints/
├── pipeline_v1.pkl        # Initial optimization
├── pipeline_v2.pkl        # Re-optimized with new data
├── pipeline_v3.pkl        # Tuned hyperparameters
└── MANIFEST.md

deployments/
├── prod-v1.pkl            # Currently in production
├── staging-v3.pkl         # Testing new version
└── rollback-v0.pkl        # Previous production
```

### Version Info File
```json
{
  "versions": [
    {
      "version": "v1.0",
      "timestamp": "2026-02-07T14:00:00Z",
      "teleprompter": "BootstrapFewShot",
      "training_data_size": 500,
      "dev_score": 0.624,
      "notes": "Initial optimization"
    },
    {
      "version": "v1.1",
      "timestamp": "2026-02-08T10:30:00Z",
      "teleprompter": "COPRO",
      "training_data_size": 750,
      "dev_score": 0.671,
      "notes": "Re-optimized with more examples"
    }
  ],
  "active": "v1.1"
}
```

## Step 5: Deployment

### Local Deployment
```bash
# Run inference server
python api.py --port 8000

# Test
curl -X POST http://localhost:8000/predict \
  -H "Content-Type: application/json" \
  -d '{"question": "What is DSPy?"}'
```

## Compilation Checklist

- [ ] Optimize pipeline with teleprompter
- [ ] Validate on dev set (quality ✓)
- [ ] Extract and save state
- [ ] Benchmark latency & cost
- [ ] Generate prompts.json
- [ ] Create deployment bundle
- [ ] Write deployment guide
- [ ] Version checkpoint
- [ ] Test inference server
- [ ] Document rollback procedure

**Goal: Create production-ready, versioned deployable packages (Docker optional).**

---

## ⚠️ Why NOT to use Pickle

Pickle is **NOT RECOMMENDED** for production. Use JSON (Step 1 above) instead.

**Why Pickle is Insecure:**
- Pickle can execute arbitrary Python code during deserialization
- Never unpickle untrusted or user-provided data
- Supply chain risk: malicious pickle files can compromise your system

**Why Pickle is Incompatible:**
- Python-specific format (cannot use in other languages)
- Breaks across Python versions or package updates
- Future Python versions may remove pickle support

**Recommendation**: Use the JSON-based `dump_state()` and `load_state()` methods (see Step 1 above). They are secure, language-agnostic, and compatible across versions.
