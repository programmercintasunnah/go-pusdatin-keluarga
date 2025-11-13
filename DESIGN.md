# Mini System Design: Weather & Chat Backend
## Handling 1 Million Requests per Day

### 📊 Traffic Analysis
- **Target**: 1,000,000 requests/day
- **Average**: ~11.6 req/sec
- **Peak Hours** (assume 8 AM - 10 PM): ~20-30 req/sec
- **Peak Load** (assume 3x average): ~60 req/sec

---

## 🏗️ Architecture Overview

```
                        ┌─────────────┐
                        │   Route 53  │ (DNS)
                        │  CloudFlare │ (DDoS Protection)
                        └──────┬──────┘
                               │
                    ┌──────────▼──────────┐
                    │   Load Balancer     │
                    │  (Nginx/HAProxy)    │
                    │  - SSL Termination  │
                    │  - Rate Limiting    │
                    └──────────┬──────────┘
                               │
            ┌──────────────────┼──────────────────┐
            │                  │                  │
     ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐
     │  Backend 1  │   │  Backend 2  │   │  Backend N  │
     │   (Golang)  │   │   (Golang)  │   │   (Golang)  │
     │  + WebSocket│   │  + WebSocket│   │  + WebSocket│
     └──────┬──────┘   └──────┬──────┘   └──────┬──────┘
            │                  │                  │
            └──────────────────┼──────────────────┘
                               │
          ┌────────────────────┼────────────────────┐
          │                    │                    │
    ┌─────▼─────┐      ┌──────▼──────┐     ┌──────▼──────┐
    │   Redis   │      │  PostgreSQL │     │    NATS     │
    │  (Cache)  │      │  (Primary)  │     │  (PubSub)   │
    │           │      │      +      │     │             │
    │  - Weather│      │  Read       │     │  - Chat     │
    │  - Session│      │  Replicas   │     │  - Events   │
    └───────────┘      └─────────────┘     └─────────────┘
```

---

## 🎯 Komponen Detail

### 1. **Load Balancer Layer**
**Nginx / AWS ALB / GCP Load Balancer**

**Config:**
```nginx
upstream backend {
    least_conn;  # Distribusi berdasarkan koneksi aktif
    server backend1:8080 max_fails=3 fail_timeout=30s;
    server backend2:8080 max_fails=3 fail_timeout=30s;
    server backend3:8080 max_fails=3 fail_timeout=30s;
}

# Rate limiting
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=ws:10m rate=5r/s;

server {
    listen 443 ssl http2;
    
    # Rate limit untuk REST API
    location /api/ {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://backend;
    }
    
    # WebSocket dengan sticky session
    location /ws {
        limit_req zone=ws burst=10;
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # Sticky session untuk WebSocket
        ip_hash;
    }
}
```

**Features:**
- SSL/TLS termination
- Rate limiting per IP
- Health checks
- Sticky sessions untuk WebSocket
- Request buffering

---

### 2. **Backend Application (Golang)**
**Stateless & Horizontally Scalable**

**Optimization:**
```go
// Connection pooling
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)

// Graceful shutdown
srv := &http.Server{
    Addr:         ":8080",
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  120 * time.Second,
}
```

**Features:**
- Goroutine untuk concurrent requests
- Connection pooling ke database
- Graceful shutdown
- Circuit breaker untuk external API
- Request timeout & context cancellation
- Structured logging

**Scaling Strategy:**
- Minimum: 2 instances (HA)
- Target: 3-5 instances untuk 1M req/day
- Auto-scale based on CPU > 70% atau memory > 80%

---

### 3. **Database Layer**

#### **PostgreSQL Primary-Replica Setup**

```
┌─────────────┐          ┌─────────────┐
│  Primary    │◄────────►│  Replica 1  │
│  (Write)    │          │   (Read)    │
└─────────────┘          └─────────────┘
       │                        │
       │                 ┌─────────────┐
       └────────────────►│  Replica 2  │
                         │   (Read)    │
                         └─────────────┘
```

**Optimization:**
```sql
-- Index yang WAJIB ada
CREATE INDEX idx_weather_collected_at ON weather_data(collected_at DESC);
CREATE INDEX idx_weather_city ON weather_data(city);
CREATE INDEX idx_weather_city_time ON weather_data(city, collected_at DESC);

CREATE INDEX idx_chat_sent_at ON chat_messages(sent_at DESC);
CREATE INDEX idx_chat_username ON chat_messages(username);

-- Partitioning untuk time-series data
CREATE TABLE weather_data (
    id SERIAL,
    city VARCHAR(100),
    temperature FLOAT,
    weather_desc TEXT,
    collected_at TIMESTAMP NOT NULL
) PARTITION BY RANGE (collected_at);

-- Partition per bulan
CREATE TABLE weather_data_2025_11 PARTITION OF weather_data
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
```

**Connection Strategy:**
- Write → Primary only
- Read (history, analytics) → Replicas (round-robin)
- Connection pool: 20-50 per backend instance

**Backup Strategy:**
- Automated daily backup
- Point-in-time recovery (PITR)
- Retention: 30 days

---

### 4. **Cache Layer (Redis)**

**Use Cases:**
```redis
# Cache current weather (TTL 10 menit)
SET weather:jakarta:current "{...}" EX 600

# Cache weather history (TTL 1 jam)
SET weather:jakarta:history:page1 "[...]" EX 3600

# Session store untuk WebSocket
HSET ws:sessions:conn123 "username" "zakie"
HSET ws:sessions:conn123 "groups" "developers,general"

# Rate limiting
INCR ratelimit:192.168.1.1:api
EXPIRE ratelimit:192.168.1.1:api 60
```

**Configuration:**
```
# Redis Cluster atau Sentinel untuk HA
maxmemory 2gb
maxmemory-policy allkeys-lru
```

**Benefits:**
- Reduce database load by 70-80%
- Sub-millisecond response time
- Session persistence untuk sticky WebSocket

---

### 5. **Message Broker (NATS)**

**Why NATS?**
- Lightweight & fast (1M+ msg/sec)
- Built-in clustering
- Low latency (<1ms)
- Better than RabbitMQ untuk real-time chat

**Architecture:**
```
Backend 1 ──┐
            ├──► NATS ──┐
Backend 2 ──┤           ├──► Subscribers
            ├──► Cluster│   (All backends)
Backend 3 ──┘           │
                        └──► WebSocket clients
```

**Usage:**
```go
// Publish chat message
nc.Publish("chat.broadcast", msgBytes)
nc.Publish("chat.private.user123", msgBytes)
nc.Publish("chat.group.developers", msgBytes)

// Subscribe to messages
nc.Subscribe("chat.*", handleMessage)
```

**Alternative:** Redis Pub/Sub (jika sudah ada Redis)

---

### 6. **External API Integration**

#### **Weather API Ingestion**

**Problem:** Open-Meteo API limit & latency

**Solution:**
```
┌──────────────┐
│  Cron Job    │ (Every 15 minutes)
│  (Separate)  │
└──────┬───────┘
       │
       ├──► Fetch weather data
       │
       ├──► Store to DB
       │
       └──► Update Redis cache
```

**Implementation:**
- Separate service untuk ingest (bukan di backend utama)
- Retry dengan exponential backoff
- Circuit breaker jika API down
- Fallback ke cached data

```go
// Circuit breaker
breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "weather-api",
    MaxRequests: 3,
    Interval:    time.Minute,
    Timeout:     30 * time.Second,
})
```

---

## 🚀 Scaling Strategy

### **Horizontal Scaling (Recommended)**

| Load Level | Backend Instances | Database | Redis | NATS |
|------------|-------------------|----------|-------|------|
| Low (< 10 req/s) | 2 | 1 Primary | 1 | 1 |
| Medium (10-30 req/s) | 3-5 | 1 Primary + 1 Replica | 1 | 3-node cluster |
| High (30-60 req/s) | 5-10 | 1 Primary + 2 Replicas | Redis Cluster | 3-node cluster |

### **Auto-scaling Rules (Kubernetes HPA)**

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: backend-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: backend
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

---

## 🔐 Security

### **1. API Security**
- Rate limiting per IP/user
- API key authentication (optional)
- CORS policy
- Input validation & sanitization
- SQL injection prevention (parameterized queries)

### **2. WebSocket Security**
- Origin validation
- Token-based authentication
- Message size limits
- Connection limits per IP

### **3. Infrastructure**
- WAF (Web Application Firewall)
- DDoS protection (CloudFlare)
- VPC/Private networking
- Secrets management (Vault/AWS Secrets Manager)

---

## 📊 Monitoring & Observability

### **Metrics to Track**

**Application Metrics (Prometheus):**
```
# Request metrics
http_requests_total{method="GET", endpoint="/api/weather/current"}
http_request_duration_seconds

# WebSocket metrics
websocket_connections_active
websocket_messages_sent_total
websocket_messages_received_total

# Database metrics
db_query_duration_seconds
db_connections_active
db_connections_idle

# Cache metrics
redis_cache_hit_rate
redis_memory_used_bytes
```

**Infrastructure Metrics:**
- CPU, Memory, Disk usage
- Network throughput
- Pod/Container health

### **Logging Strategy**

```go
// Structured logging dengan context
log.WithFields(log.Fields{
    "request_id": ctx.Value("request_id"),
    "user_id":    user.ID,
    "endpoint":   "/api/weather/current",
    "latency_ms": latency,
    "status":     200,
}).Info("Request completed")
```

**Log Aggregation:**
- ELK Stack (Elasticsearch, Logstash, Kibana)
- Or: Loki + Grafana
- Or: CloudWatch Logs (AWS)

### **Alerting**

```yaml
# Prometheus AlertManager rules
groups:
- name: backend_alerts
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
    annotations:
      summary: "High 5xx error rate"
      
  - alert: DatabaseConnectionPoolExhausted
    expr: db_connections_active >= db_connections_max * 0.9
    annotations:
      summary: "Database connection pool nearly exhausted"
      
  - alert: HighLatency
    expr: histogram_quantile(0.95, http_request_duration_seconds) > 1
    annotations:
      summary: "95th percentile latency > 1s"
```

---

## 💰 Cost Estimation (AWS/GCP)

### **Monthly Cost Breakdown**

| Component | Specification | Monthly Cost |
|-----------|--------------|--------------|
| Load Balancer | ALB/NLB | $20-30 |
| Backend (3x) | t3.medium (2 vCPU, 4GB) | $90 |
| PostgreSQL | db.t3.medium + replicas | $120 |
| Redis | cache.t3.small | $30 |
| NATS | self-hosted on t3.small | $15 |
| Bandwidth | ~500GB/month | $45 |
| Monitoring | Prometheus + Grafana (self-hosted) | $15 |
| **Total** | | **~$335/month** |

**Cloud-Managed Alternative:**
- AWS RDS PostgreSQL: +$50
- AWS ElastiCache Redis: +$20
- Total: ~$405/month

---

## 🎯 Performance Optimization

### **Database Query Optimization**

```sql
-- BAD: Full table scan
SELECT * FROM weather_data WHERE city = 'Jakarta' ORDER BY collected_at DESC;

-- GOOD: Use index
SELECT * FROM weather_data 
WHERE city = 'Jakarta' 
ORDER BY collected_at DESC 
LIMIT 10;

-- Use EXPLAIN ANALYZE
EXPLAIN ANALYZE SELECT ...;
```

### **Caching Strategy**

```
Request → Check Redis → If hit: return
                     → If miss: Query DB → Store in Redis → return
```

**Cache Invalidation:**
- Time-based (TTL)
- Event-based (on data update)
- Lazy update (update on next request)

### **Connection Pooling**

```go
// PostgreSQL
db.SetMaxOpenConns(25)        // Max connections
db.SetMaxIdleConns(10)        // Keep 10 idle
db.SetConnMaxLifetime(5 * time.Minute)

// Redis
&redis.Options{
    PoolSize:     10,
    MinIdleConns: 3,
}
```

---

## 🧪 Load Testing

**Before Production:**

```bash
# Apache Bench
ab -n 10000 -c 100 http://localhost:8080/api/weather/current

# k6 (recommended)
k6 run --vus 100 --duration 30s load-test.js

# Expected results untuk 1M req/day:
# - P95 latency < 500ms
# - P99 latency < 1s
# - Error rate < 0.1%
```

---

## 📝 Deployment Strategy

### **Blue-Green Deployment**

```
┌─────────────┐         ┌─────────────┐
│   Blue      │         │   Green     │
│  (Current)  │◄───────►│   (New)     │
│  Version 1  │         │  Version 2  │
└─────────────┘         └─────────────┘
      ▲                        │
      │                        │
      └──── Switch traffic ────┘
```

### **Rolling Update (Kubernetes)**

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0
```

---

## ✅ Checklist Production-Ready

- [ ] Load balancer dengan health checks
- [ ] Minimum 2 backend instances (HA)
- [ ] Database dengan backup otomatis
- [ ] Redis untuk caching
- [ ] Rate limiting di load balancer
- [ ] Monitoring & alerting setup
- [ ] Logging aggregation
- [ ] SSL/TLS certificate
- [ ] DDoS protection
- [ ] Auto-scaling configured
- [ ] Load testing completed
- [ ] Disaster recovery plan
- [ ] Documentation lengkap

---

## 🚨 Disaster Recovery

**RTO (Recovery Time Objective):** < 15 minutes  
**RPO (Recovery Point Objective):** < 5 minutes

**Backup Strategy:**
- Database: Daily full + hourly incremental
- Config: Version controlled (Git)
- Secrets: Encrypted vault

**Failover Plan:**
1. Automated health checks
2. Automatic failover ke replica
3. Alert ops team
4. Manual verification
5. Post-mortem analysis

---

## 📈 Future Improvements

### **When Scale > 10M req/day:**
1. **CDN** untuk static content
2. **GraphQL** untuk efficient data fetching
3. **Microservices** (split weather & chat)
4. **Event Sourcing** untuk chat history
5. **Multi-region deployment**
6. **Kafka** untuk event streaming
7. **Elasticsearch** untuk chat search

### **Advanced Features:**
- WebSocket connection pooling
- Message queue untuk async processing
- Read-through/Write-through cache
- Database sharding
- Global load balancing