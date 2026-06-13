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
   pkg/        (재사용 가능 - 다른 모듈에서 import 가능)
   ├─ kube       K8s REST → pod image tag, startTime
   ├─ argocd     Application API → repo URL, synced revision(SHA) + Sync(rollback)
   ├─ github     synced SHA → /commits/{sha} → 커미터 이름/이메일
   ├─ slackapi   lookupByEmail / chat.postMessage / views.open
   ├─ grafana    Silences API (silence 모달 제출)
   ├─ httpx      공유 JSON/HTTP 헬퍼
   └─ model      공유 타입 (Alert, EnrichedAlert)
   internal/   (앱 전용 - 오케스트레이션 글루)
   ├─ webhook    Grafana webhook 파싱 + Slack 서명검증
   ├─ enricher   소스 조립 + graceful degrade
   ├─ notifier   메인/댓글/액션 Block Kit + 발송
   └─ config     중앙 설정 로더
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

## 재사용 (다른 프로젝트에서 import)

`pkg/` 아래 클라이언트는 의존성 0(stdlib만)이라 어느 환경에서도 그대로 가져다 쓸 수 있다.

```go
import (
    "github.com/kdh8733/monitoring/pkg/argocd"
    "github.com/kdh8733/monitoring/pkg/grafana"
)

argo := argocd.New("https://argocd.example.com", token)
info, _ := argo.AppInfo(ctx, "checkout-api")   // repo URL, synced revision, prev revision

g := grafana.New("https://grafana.example.com", token)
id, _ := g.CreateSilence(ctx, matchers, time.Hour, "me", "deploy issue")
```

각 클라이언트는 `New(baseURL, token)` 한 가지 패턴이고, enricher가 의존하는 부분은
인터페이스(`KubeSource`/`ArgoSource`/`GitHubSource`/`SlackIdentity`)로 추상화돼 있어
다른 구현(공식 SDK 기반 등)으로 갈아끼우기 쉽다. `internal/`은 이 앱 전용 글루라
import 대상이 아니다.
