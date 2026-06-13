# 공식 Go SDK 전환 트레이드오프

현재 `pkg/` 클라이언트는 stdlib REST로 짜여 있다(의존성 0). 공식 Go SDK로 바꿀지
통합별로 판단한 기록. enricher가 인터페이스(`KubeSource`/`ArgoSource`/`GitHubSource`/
`SlackIdentity`)에 의존하므로, **앱 로직을 건드리지 않고 `pkg/` 구현만 통합별로 교체**할 수 있다.

## 일반 트레이드오프

| | REST(현재) | 공식 SDK |
|---|---|---|
| 의존성 | 0 (stdlib만) | 큼 - 특히 client-go/argo는 수백 개 transitive dep |
| 타입 안전성 | 직접 struct 정의(부분 필드) | 전체 typed object, 컴파일 타임 안전 |
| 인증/엣지 | 수동(Bearer 등) | kubeconfig/in-cluster/페이지네이션/재시도 내장 |
| 빌드/이미지 | 빠름, 작음 | 느림, 큼 |
| 버전 결합 | 없음 | argo/k8s는 SDK 버전과 클러스터 버전 정렬 필요 |
| 스트리밍 | 없음 | watch/informer(k8s) 등 가능 |
| import 마찰 | 없음(skeleton 강점) | 큼(transitive 충돌 가능) |

## 통합별 권장

| 통합 | 후보 SDK | 권장 | 이유 |
|---|---|---|---|
| Kubernetes | `k8s.io/client-go` | **전환 권장** | kubeconfig/in-cluster 인증 + typed pod + 향후 watch까지 REST로 직접 하기엔 비용이 큼. K8s가 SDK 이득이 가장 분명한 영역. |
| Slack | `slack-go/slack` | **고려할 만함** | Block Kit typed 빌더 + 서명검증 헬퍼 + lookupByEmail/postMessage/views.open 전부 커버. dep 중간. 현재 hand-rolled Block Kit map[string]any를 타입으로 대체 가능. |
| GitHub | `google/go-github` | 선택(낮은 우선) | 단일 엔드포인트(`/commits/{sha}`)만 쓰므로 REST로 충분. 페이지네이션/다수 API 쓰게 되면 전환. |
| ArgoCD | `argoproj/argo-cd/v2` | **REST 유지** | Go client가 k8s deps까지 끌고 와 매우 무겁고 argo 버전에 강결합. REST API는 안정적이고 우리가 쓰는 범위(app get/sync)는 작음. |
| Grafana | (공식 부재, 커뮤니티 OpenAPI) | **REST 유지** | 강력한 공식 Go SDK가 없음. Silences API REST로 충분. |
| Prometheus | `prometheus/client_golang` api/v1 | 필요 시만 | 현재 Prometheus를 직접 쿼리하지 않음(Grafana가 함). 직접 PromQL 쿼리를 붙일 때 도입(상대적으로 가벼움). |

## 결론

- 우선순위: **K8s(client-go) > Slack(slack-go) > 나머지 REST 유지**.
- 무거운 SDK(client-go, argo client) 도입 시 "어느 환경에서나 import" 강점이 약해지므로,
  재사용 skeleton 성격을 유지하려면 **선택적**으로만 바꾼다.
- 교체는 통합별로 독립 가능(인터페이스 경계). 예: `pkg/kube`만 client-go 구현으로
  바꾸고 `KubeSource`를 그대로 만족시키면 enricher/notifier/cmd는 무변경.
