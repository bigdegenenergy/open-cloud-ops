# CoreNLP vs Stanza: Detailed Comparison

## Architecture

### CoreNLP (Java)
- **Model**: Rule-based + neural
- **Pipeline**: Sequential annotation processors
- **Built**: Stanford NLP Group (2010s)
- **Approach**: Established, battle-tested

### Stanza (Python)
- **Model**: Neural (PyTorch)
- **Pipeline**: Pre-trained transformers
- **Built**: Stanford NLP Group (2020s)
- **Approach**: Modern, research-focused

## Task Performance

### Tokenization
| Task | CoreNLP | Stanza | Winner |
|---|---|---|---|
| English | Excellent | Excellent | Tie |
| Chinese | Good | Excellent | Stanza |
| Mixed script | Good | Excellent | Stanza |

### Sentence Segmentation
| Task | CoreNLP | Stanza | Winner |
|---|---|---|---|
| English | Excellent | Excellent | Tie |
| With abbreviations | Very Good | Excellent | Stanza |
| Multiple languages | Limited | Excellent | Stanza |

### POS Tagging
| Task | CoreNLP | Stanza | Winner |
|---|---|---|---|
| English | Excellent | Excellent | Tie |
| Morphologically rich | Good | Excellent | Stanza |
| Multiple languages | Limited | Excellent | Stanza |

### Named Entity Recognition
| Task | CoreNLP | Stanza | Winner |
|---|---|---|---|
| English news | Excellent | Excellent | Tie |
| Domain adaptation | Fair | Good | Stanza |
| Low-resource language | Fair | Good | Stanza |

### Dependency Parsing
| Task | CoreNLP | Stanza | Winner |
|---|---|---|---|
| English (UD) | Excellent | Excellent | Tie |
| Multiple languages | Limited | Excellent | Stanza |
| Speed | Excellent | Good | CoreNLP |

### Coreference Resolution
| Task | CoreNLP | Stanza | Winner |
|---|---|---|---|
| English | Excellent | Emerging | CoreNLP |
| Maturity | Mature | Developing | CoreNLP |
| Research use | Established | Growing | CoreNLP |

## Language Support

### CoreNLP
**6 languages:**
- English ✅ (full support)
- Chinese ✅ (full support)
- German ✅ (full support)
- Spanish ✅ (good support)
- French ✅ (good support)
- Arabic ✅ (limited support)

### Stanza
**70+ languages:**
- All above + many more
- Includes: Japanese, Korean, Vietnamese, Thai, Hebrew, Polish, Russian, Turkish, Ukrainian, Persian, Finnish, Swedish, Norwegian, Dutch, Greek, Czech, Slovak, Romanian, Bulgarian, Croatian, Serbian, Slovenian, Estonian, Latvian, Lithuanian, Hungarian, Albanian, Marathi, Tamil, Telugu, Urdu, Vietnamese, Yoruba, Swahili, and more

**Winner: Stanza** (70+ vs 6)

## Speed & Resources

### Startup Time
- **CoreNLP**: 5-10 seconds (first run with models)
- **Stanza**: 2-5 seconds
- **Winner: Stanza**

### Memory Usage
- **CoreNLP**: 1-2 GB (all models in memory)
- **Stanza**: 500 MB - 2 GB (depending on processors)
- **Winner: Stanza** (slightly more memory efficient)

### Processing Speed (tokens/second)
| Task | CoreNLP | Stanza | Winner |
|---|---|---|---|
| Tokenization | 50K+ | 30K | CoreNLP |
| Full pipeline | 5K | 3K | CoreNLP |
| CPU | Faster | Slower | CoreNLP |
| GPU | N/A | Excellent | Stanza |

## Integration

### In Java Projects
```
CoreNLP: Native, embedded in code
Stanza: Need language bridge (jython, subprocess)
```
**Winner: CoreNLP** (natural Java integration)

### In Python Projects
```
CoreNLP: Need jpype or subprocess
Stanza: Native Python
```
**Winner: Stanza** (native Python integration)

### REST API
```
CoreNLP: Built-in server support
Stanza: Flask/FastAPI wrappers needed
```
**Winner: Tie** (both doable, CoreNLP easier)

## Customization

### CoreNLP
- Modify rules
- Train new models (MATLAB)
- Limited to provided annotations

### Stanza
- Fine-tune transformers (PyTorch)
- Add custom layers
- More flexibility for research

**Winner: Stanza** (easier to extend)

## Documentation

### CoreNLP
- Official documentation: Comprehensive
- Examples: Many
- Community: Active but aging
- Research papers: Yes

### Stanza
- Official documentation: Good
- Examples: Good
- Community: Growing
- Research papers: Recent

**Winner: Tie** (both well documented)

## Cost & Licensing

### CoreNLP
- **License**: GPL 3.0 (commercial support available)
- **Cost**: Free (or commercial license)
- **Models**: Included
- **Note**: Some models may have additional restrictions depending on training data source

### Stanza
- **License**: Apache 2.0 (code)
- **Cost**: Free
- **Models**: Downloaded on demand
- **Note**: Pre-trained models vary by license. Models trained on LDC data (CoNLL, Universal Dependencies) may be restricted to non-commercial use. Always verify model-specific licenses.

**Winner: Stanza** (more permissive code license, but verify model licenses)

⚠️ **Important**: Even if framework code is open-source, individual models may have restrictions. Check model documentation before deploying.

## Ecosystem

### CoreNLP
- Integrated with: Java frameworks, enterprise tools
- Plugins: Limited
- Extensions: Maven repos

### Stanza
- Integrated with: PyTorch, Hugging Face, scikit-learn
- Fine-tuning: Supported
- Community models: Sharing available

**Winner: Stanza** (stronger modern ML ecosystem)

## Decision Tree

```
┌─ Do you have a Java codebase?
│  ├─ Yes, production?  → CoreNLP ✅
│  ├─ Yes, flexible?    → Stanza (with bridge) or CoreNLP
│  └─ No               ↓
│
├─ Do you need 70+ languages?
│  ├─ Yes              → Stanza ✅
│  ├─ No, just English → Tie (either works)
│  └─ 6 languages fine → CoreNLP ✅
│
├─ Do you need coreference resolution?
│  ├─ Yes, mature      → CoreNLP ✅
│  ├─ Yes, research ok → Either
│  └─ No              ↓
│
├─ Do you have a GPU?
│  ├─ Yes              → Stanza ✅ (GPU accelerated)
│  └─ No              ↓
│
├─ Is speed critical (real-time)?
│  ├─ Yes              → CoreNLP ✅
│  ├─ Batch ok         → Either
│  └─ Research ok      → Stanza ✅
│
└─ Do you need to fine-tune models?
   ├─ Yes              → Stanza ✅
   └─ No, out-of-box   → Either
```

## Recommendations

### Use CoreNLP if:
✅ Java production system
✅ Coreference resolution needed
✅ Speed is critical
✅ Need proven, mature tools
✅ Enterprise support required

### Use Stanza if:
✅ Python project
✅ Multi-lingual support (20+ languages)
✅ Have GPU available
✅ Need to fine-tune models
✅ Research or rapid prototyping
✅ Modern ML ecosystem needed

## Hybrid Approach

For maximum flexibility:
```python
# Python project needing coreference?
# Use Stanza for most tasks, bridge to CoreNLP for coreference

import stanza
from py4j.java_gateway import JavaGateway

nlp_stanza = stanza.Pipeline('en')  # Tokenize, parse, NER
gateway = JavaGateway()  # CoreNLP for coreference

doc_stanza = nlp_stanza(text)
# ...use Stanza results...

doc_corenlp = gateway.entry_point.process(text)
# ...get coreference chains...
```

## Conclusion

**For most projects**, the choice is straightforward:
- **Java + Production?** CoreNLP
- **Python + Modern?** Stanza
- **Multi-lingual?** Stanza
- **Coreference?** CoreNLP
- **Research?** Stanza

Both are excellent. Pick the one that fits your constraints.
