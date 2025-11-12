# Mini System Design: Weather & Chat Backend

## Goal
Handle ±1.000.000 requests per day (~11.5 requests/sec)

## Komponen
1. **Load Balancer**
   - Gunakan Nginx / Cloud Load Balancer
   - Mendistribusikan trafik ke beberapa instance backend

2. **Backend (Golang)**
   - Stateless → mudah di-scale horizontal
   - Gunakan goroutine untuk koneksi WebSocket/chat concurrency

3. **Database**
   - PostgreSQL + Index di `collected_at` & `city`
   - Gunakan connection pool (max 20–50)

4. **Cache Layer**
   - Redis untuk cache data cuaca terbaru (mengurangi query DB)

5. **Message Broker (opsional)**
   - Gunakan NATS / RabbitMQ untuk broadcast chat ke beberapa node

6. **Autoscaling**
   - Deploy di container cluster (Docker Swarm / Kubernetes)
   - Scale berdasarkan CPU atau request throughput

7. **Monitoring**
   - Prometheus + Grafana
   - Log agregator (ELK stack)

