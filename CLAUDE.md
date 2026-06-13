# Monitoring - 프로젝트 한정 규칙

글로벌 `~/.claude/CLAUDE.md`를 상속한다. 아래는 이 프로젝트에서만 추가되는 차이점.

## 확정된 설계 결정 (재논의 금지 - 변경하려면 ROADMAP 갱신부터)

- 알람 엔진 = **Grafana Alerting** 단일 contact point. Silence = Grafana-managed
  Alertmanager Silences API. (AlertManager 직접 연동 아님)
- 멀티클러스터 = **중앙 집중형**. cluster name은 alert 라벨로 식별, Go 앱은 중앙
  Grafana 1곳만 상대.
- 배포 책임자 = **ArgoCD synced revision(SHA)의 커미터**. repo HEAD 마지막 커밋 아님.
- ArgoCD rollback 등 액션은 `ARGOCD_ROLLBACK_ENABLED` 플래그 게이트.

## Go 작업 규율

- 기능/버그는 글로벌 TDD 원칙대로 **테스트 먼저**. 외부(K8s/Grafana/GitHub/Slack/ArgoCD)는
  인터페이스로 추상화하고 fake로 단위테스트. 네트워크 호출을 테스트에 넣지 않는다.
- 커밋 전 `gofmt`/`go vet` 통과. (golangci-lint 도입 시 CI에서 강제)
- Go는 `/usr/local/go`(go1.26)에 설치됨. 외부 의존성 0(stdlib만) - `go test ./...`가
  네트워크 없이 돈다. 새 의존성 추가는 이 원칙을 깨는지 먼저 따져본다.
- 모든 외부 연동은 `{BaseURL, Token}` 패턴 + REST(net/http). 무거운 SDK(client-go/slack-go)
  대신 얇은 클라이언트 + 인터페이스 + fake 테스트.

## 시크릿

- `.env`, 토큰, kubeconfig는 절대 커밋하지 않는다(`.gitignore` 등록됨). 예시는
  `.env.example`에만.
