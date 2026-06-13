# 배포 & 연동 가이드

## 빌드 / 배포

```bash
docker build -t <registry>/monitoring:latest .
docker push <registry>/monitoring:latest

kubectl create namespace monitoring
# 중앙 설정 Secret 생성 (.env 채운 뒤)
kubectl -n monitoring create secret generic monitoring-config --from-env-file=.env
# 또는 deploy/secret.example.yaml 복사해 채운 뒤 apply

# deployment.yaml 의 image: REPLACE_ME 교체 후
kubectl apply -f deploy/deployment.yaml
```

엔드포인트(인그레스 뒤): `https://<host>/healthz`, `/webhook/grafana`, `/slack/interactivity`.

## 외부 플랫폼 연동 체크리스트

각 항목의 값은 모두 `.env`(또는 monitoring-config Secret) 한 곳에 넣는다.

### 1. Grafana Alerting (알람 엔진)
- Contact point: **Webhook** 타입, URL = `https://<host>/webhook/grafana`.
- Notification policy를 이 contact point로 라우팅.
- 알람 rule 라벨에 `cluster`, `namespace`, `app`(또는 service)가 실리도록 구성.
- `GRAFANA_BASE_URL`, `GRAFANA_API_TOKEN`(Silences 쓰기 권한) 설정 -> Silence 모달용.

### 2. Slack App
- **Interactivity & Shortcuts** ON -> Request URL = `https://<host>/slack/interactivity`.
- Bot Token Scopes: `chat:write`, `users:read.email`.
- 봇을 알람 채널에 초대. `SLACK_ALERT_CHANNEL` = 채널 ID.
- `SLACK_BOT_TOKEN`(xoxb-...), `SLACK_SIGNING_SECRET`(Basic Information) 설정.

### 3. Kubernetes (중앙 API 접근)
- 대상 네임스페이스 pod read 권한의 ServiceAccount 토큰 발급.
- `KUBE_API_URL` = API 서버 주소, `KUBE_TOKEN` = SA 토큰.
- 멀티클러스터는 중앙 API가 cluster 라벨로 구분(중앙 집중형 전제).

### 4. GitHub
- `GITHUB_TOKEN` = PAT(대상 repo의 contents read). GHE면 `GITHUB_BASE_URL` 지정.

### 5. ArgoCD
- `ARGOCD_BASE_URL`, `ARGOCD_TOKEN`(app get 권한; rollback 쓰려면 sync 권한).
- Slack rollback 버튼은 `ARGOCD_ROLLBACK_ENABLED=true`일 때만 노출/동작.

### 6. 로그 원천
- `LOG_SOURCE=kibana|prometheus` + 해당 `KIBANA_BASE_URL` 또는 `PROMETHEUS_BASE_URL`.
