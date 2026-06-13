# Monitoring

> Grafana Alerting이 firing한 알람을 받아, 알람 시점에 K8s/ArgoCD/GitHub/Slack를
> 라이브 조회해 정보를 붙이고, Slack 스레드로 발송·상호작용하는 Go 알람 오케스트레이터.

Prometheus annotation에 의존하지 않는다. 알람이 오면 그 시점에 동적으로 조회해서 붙인다.

## 아키텍처

```
중앙 Prometheus(멀티클러스터, cluster 라벨)
        │  datasource query
        ▼
중앙 Grafana Alerting  ── rule 평가 → firing
        │  contact point: webhook
        ▼
[Monitoring (Go) - 알람 오케스트레이터]
   ├─ webhook/    Grafana webhook 수신 + Slack interactivity 수신(서명검증)
   ├─ enricher/
   │   ├─ kubernetes.go  client-go → pod image tag, startTime, cluster
   │   ├─ argocd.go      Application API → repo URL, synced revision(SHA), (rollback 트리거: 플래그)
   │   ├─ github.go      synced SHA → /commits/{sha} → 커미터 이름/이메일
   │   └─ slack.go       이메일 → users.lookupByEmail → @멘션 user ID
   ├─ grafana/    패널/로그 링크, alert rule, Silences API(silence 모달)
   └─ notifier/   메인 스레드 + 스레드 댓글(Block Kit) Slack 발송
        │
        ▼
Slack
   메인 스레드: cluster / app(service) / 배포 repo + 이슈 요약
   스레드 댓글:
     1. Silence 버튼 → Block Kit 모달 → Grafana Silences API
     2. 패널 링크 + 로그 원천(Kibana 또는 Prometheus) + alert rule
     3. 배포 책임자 @멘션 (ArgoCD synced revision의 커미터)
     4. (옵션, 플래그) ArgoCD rollback 버튼
```

### 핵심 결정

- **알람 엔진 = Grafana Alerting** (단일 contact point). Silence도 Grafana-managed
  Alertmanager Silences API로 일원화. (기존 노트의 AlertManager 가정에서 변경)
- **멀티클러스터 = 중앙 집중형**. cluster name은 alert 라벨로 식별. Go 앱은 중앙
  Grafana 1곳만 상대한다.
- **배포 책임자 = ArgoCD synced revision의 커미터**. repo HEAD의 마지막 커밋이 아니라
  "실제 돌고 있는 버전을 배포한 사람"을 태그한다. (image tag 파싱보다 ArgoCD revision이 정확)
- **ArgoCD 액션(rollback)** 은 `ARGOCD_ROLLBACK_ENABLED` 플래그로 on/off.

## 빠른 시작

```bash
go version            # 1.23+ (개발 머신에 go1.26 설치됨)
go test ./...         # 전 패키지 단위테스트
cp .env.example .env  # 중앙 설정 파일 - 값만 채우면 됨 (공란이어도 기동)
go run ./cmd/server   # :8080, curl localhost:8080/healthz -> ok
```

설정은 전부 [.env.example](.env.example) 한 파일에서 관리한다(중앙 설정).
컨테이너/배포는 [deploy/](deploy/README.md), 마일스톤은 [ROADMAP.md](ROADMAP.md) 참조.
