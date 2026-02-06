# ADR-002: Privacy-First Design for LLM Proxy

**Status:** Accepted
**Date:** 2026-02-06
**Deciders:** Open Cloud Ops Core Team
**Category:** Security / Privacy

---

## Context

Cerebra, the LLM Gateway module, operates as a reverse proxy that sits between consumers (AI agents, applications, developers) and upstream LLM providers (OpenAI, Anthropic, Google Gemini). Every LLM API request and response passes through Cerebra, which means the system has access to:

- **Prompt content** -- potentially containing proprietary code, business logic, customer data, PII, medical records, legal documents, or other sensitive information
- **Response content** -- LLM-generated outputs that may contain or reference sensitive input data
- **API keys** -- credentials that grant access to paid LLM services and could be exploited if leaked
- **Usage metadata** -- model names, token counts, timestamps, agent identifiers, and computed costs

This creates a significant trust boundary. Organizations adopting Cerebra must trust that the proxy does not exfiltrate, store, or expose their sensitive data. In the event of a database breach, the blast radius must be minimized.

### Industry Context

Several high-profile incidents have demonstrated the risks of storing LLM interaction data:

- LLM prompt logs have been accidentally exposed through misconfigured storage buckets
- API key leaks in logs have led to unauthorized usage and unexpected billing
- Regulatory frameworks (GDPR, HIPAA, SOC 2) impose strict requirements on processing and storing user data

### Threat Model

| Threat | Description | Impact |
|--------|-------------|--------|
| Database breach | Attacker gains read access to PostgreSQL | If prompts are stored: full exposure of all user interactions. If only metadata: limited to usage patterns. |
| Log exfiltration | Attacker accesses application logs | If keys/prompts are logged: credential theft and data exposure. If only metadata: limited impact. |
| Insider threat | Malicious operator reads database | Same impact as database breach. |
| Backup exposure | Database backups stored insecurely | Same impact as database breach. |
| Memory dump | Attacker captures process memory | Transient exposure of in-flight requests only. No historical data. |

## Decision

We adopt a **privacy-first design** for the Cerebra LLM Gateway with the following rules:

### Rule 1: Never Store Prompt or Response Content

Cerebra treats prompt and response content as **opaque passthrough data**. The proxy forwards request bodies to the upstream provider and returns response bodies to the caller without inspecting, logging, or persisting the content.

The `APIRequest` model (defined in `cerebra/pkg/models/models.go`) explicitly excludes content fields:

```go
// Note: Prompt content and response content are NEVER stored.
type APIRequest struct {
    ID             string
    Provider       LLMProvider
    Model          string
    AgentID        string
    TeamID         string
    OrgID          string
    InputTokens    int64
    OutputTokens   int64
    TotalTokens    int64
    CostUSD        float64
    LatencyMs      int64
    StatusCode     int
    WasRouted      bool
    OriginalModel  string
    RoutedModel    string
    SavingsUSD     float64
    Timestamp      time.Time
}
```

There is no `Prompt`, `Response`, `RequestBody`, or `ResponseBody` field. This is a deliberate architectural constraint, not an oversight.

### Rule 2: Never Persist API Keys

API keys are passed through in-memory only during request processing. They are:

- Read from the incoming request's `Authorization` header
- Held in a local variable within the request handler's scope
- Forwarded to the upstream provider in the outgoing request's `Authorization` header
- Released from memory when the request handler returns (garbage collected)

API keys are never written to:
- The database (`api_requests` table has no key column)
- Redis cache
- Application logs (even at debug level)
- Disk (no temporary files)

### Rule 3: Log Only Metadata

The following metadata is extracted from each request/response and persisted for cost tracking and analytics:

| Field | Source | Purpose |
|-------|--------|---------|
| `provider` | Request URL path | Identify which LLM provider was called |
| `model` | Response headers or response JSON (model field only) | Identify which model was used for pricing |
| `input_tokens` | Response usage metadata | Calculate input cost |
| `output_tokens` | Response usage metadata | Calculate output cost |
| `total_tokens` | Computed | Total token consumption |
| `cost_usd` | Computed from model pricing table | Cost tracking |
| `latency_ms` | Measured (request start to response end) | Performance monitoring |
| `status_code` | Response HTTP status | Error rate tracking |
| `agent_id` | Request header (`X-Agent-ID`) | Cost attribution |
| `team_id` | Request header (`X-Team-ID`) | Cost attribution |
| `org_id` | Request header (`X-Org-ID`) | Cost attribution |
| `was_routed` | Smart router decision | Track routing effectiveness |
| `original_model` | Original request | Routing audit trail |
| `routed_model` | Router selection | Routing audit trail |
| `savings_usd` | Computed | Track savings from routing |
| `timestamp` | Server clock | Time-series indexing |

### Rule 4: Smart Router Must Not Depend on Content

The smart routing engine (`cerebra/internal/router/`) selects models based on:

- Budget constraints (remaining spend for the agent/team)
- Configured routing strategy (cost_optimized, quality_first, latency_optimized, adaptive)
- Model tier assignments (economy, standard, premium)
- Historical performance metrics

The router does **not** inspect prompt content to determine complexity. If content-based routing is added in the future, it must operate on content hashes or embeddings computed in-memory, never persisted.

## Consequences

### Positive

- **Minimized breach impact:** If the database is compromised, attackers obtain only usage metadata (which models were called, how many tokens were used, cost per request). No prompts, responses, or API keys are exposed. This dramatically reduces the severity of a breach.
- **Regulatory compliance:** By not storing content, Cerebra avoids triggering data processing obligations under GDPR (no personal data storage), HIPAA (no PHI storage), and similar regulations. The metadata stored does not constitute personal data in most jurisdictions.
- **User trust:** Organizations can adopt Cerebra with confidence that their proprietary prompts and code are not being collected. This is a key differentiator for open-source adoption.
- **Simpler data management:** No need for content encryption at rest, content retention policies, right-to-be-forgotten implementations, or content-specific access controls.
- **Reduced storage requirements:** LLM prompts and responses can be very large (tens of thousands of tokens). Storing only metadata keeps database size manageable and time-series queries fast.

### Negative

- **No prompt replay or debugging:** If an LLM request produces an unexpected result, there is no server-side record of what was sent or received. Debugging must happen on the client side. Mitigated by recommending client-side logging for development environments.
- **No content-based analytics:** Features like "show me the most common prompt patterns" or "detect prompt injection attempts" are not possible without content storage. Mitigated by providing rich metadata-based analytics (cost by model, cost by agent, token distribution, etc.).
- **No content-based routing intelligence:** The smart router cannot analyze prompt complexity by reading the prompt. Mitigated by using proxy signals (token count estimates, model tier preferences) rather than content inspection.
- **Limited audit trail:** Compliance teams may want to audit what was sent to an LLM provider. Cerebra can confirm that a request was made (metadata) but not what was in the request (content). Mitigated by recommending client-side audit logging for organizations that require it.

### Implementation Constraints

These constraints must be enforced in all future development:

1. **Code review gate:** Any PR that adds a content-storing field to `APIRequest` or any database table must be explicitly flagged and rejected unless accompanied by a new ADR superseding this one.
2. **Log sanitization:** All logging middleware must be configured to exclude request and response bodies. Only status codes, latencies, and metadata headers may be logged.
3. **Testing:** Integration tests must verify that the `api_requests` table contains no content fields after proxy operations.
4. **Documentation:** All API documentation must clearly state that content is never stored.

## Alternatives Considered

### 1. Store Content with Encryption at Rest

Encrypt prompt and response content before storing it in the database, using a customer-managed encryption key.

- **Rejected because:** Encrypted data is still data. A key compromise (or a court order compelling key disclosure) would expose all historical content. The operational complexity of key management, rotation, and per-tenant encryption outweighs the benefits. Most importantly, this violates the core principle that content storage is unnecessary for the cost-tracking use case.

### 2. Store Content with Configurable Retention

Allow users to opt in to content storage with a configurable retention period (e.g., 7 days for debugging).

- **Rejected because:** Optional content storage creates a split architecture where some deployments have content and some do not. Security audits must account for the worst case (content is stored). It also creates a false sense of security -- "only 7 days" still means 7 days of exposure in a breach. This may be reconsidered in a future ADR if strong user demand materializes, but it would require explicit opt-in, per-request consent, and encryption at rest.

### 3. Store Content Hashes Only

Store SHA-256 hashes of prompts to enable deduplication and pattern detection without storing raw content.

- **Rejected because:** Content hashes still enable correlation attacks (if an attacker knows or guesses a prompt, they can verify it was sent by comparing hashes). Hashes also provide limited analytical value without the raw content. The marginal benefit does not justify the added complexity and the weakened privacy posture.

### 4. Client-Side Content Logging

Provide an SDK that logs content on the client side, with the client choosing its own storage backend.

- **Considered and accepted as a complementary approach.** Client-side logging is recommended for organizations that need content audit trails. This keeps the decision about content storage with the data owner (the client) rather than the proxy operator.

## References

- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [GDPR Article 5: Principles relating to processing of personal data](https://gdpr.eu/article-5-how-to-process-personal-data/)
- [Data Minimization Principle](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/data-protection-principles/a-guide-to-the-data-protection-principles/the-principles/data-minimisation/)
- Cerebra model definition: `cerebra/pkg/models/models.go`
- Cerebra proxy handler: `cerebra/internal/proxy/handler.go`
