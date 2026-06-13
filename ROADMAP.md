# ROADMAP - Monitoring

> Goal와 완료 기준의 단일 원천(source of truth). 마일스톤 상세 분해/태스크 계획은
> `/project-plan` 및 superpowers `writing-plans`가 이어받는다.

## Goal

중앙 Grafana Alerting이 firing한 알람을 Go 오케스트레이터가 받아, 알람 시점에
K8s/ArgoCD/GitHub/Slack를 라이브 조회해 enrich하고, Slack 스레드(메인 + 상호작용 댓글)로
발송한다. 목적: 문제 발생 시 대상 클러스터/서비스/배포 책임자를 즉시 인지시키고,
스레드에서 바로 silence/rollback 조치를 취할 수 있게 한다.

## 완료 기준 (검증 가능 - 이게 충족되면 Goal 달성)

1. **E2E 발송**: Grafana Alerting의 테스트 알람을 `/webhook/grafana`로 받으면,
   Slack 메인 스레드에 `cluster name` + `app/service name(k8s)` + `배포 repo name`이
   포함되어 게시된다. (검증: 테스트 webhook payload → Slack 메시지 필드 assert)
2. **Silence 모달**: 스레드의 Silence 버튼 클릭 → Block Kit 모달(기간/코멘트) 제출 →
   Grafana Silences API에 silence 생성. (검증: 제출 후 Silences API GET으로 해당
   silence 존재 확인)
3. **컨텍스트 댓글**: 스레드 댓글에 대상 패널 링크 + 로그 원천 링크(Kibana 또는
   Prometheus) + alert rule 이름이 포함된다. (검증: 링크가 200/유효 라우트, rule 이름 일치)
4. **배포 책임자 태깅**: ArgoCD synced revision(SHA)의 커미터 이메일을
   `users.lookupByEmail`로 Slack user ID에 매칭해 @멘션한다. (검증: 멘션된 user ID가
   해당 커미터 이메일로 resolve됨. 매칭 실패 시 fallback 메시지로 graceful degrade)
5. **rollback 플래그**: rollback 버튼은 `ARGOCD_ROLLBACK_ENABLED=true`일 때만
   렌더/처리되고, 클릭 시 ArgoCD Application을 직전 revision으로 sync한다.
   (검증: 플래그 off면 버튼 없음 / on이면 sync 호출됨)

## 마일스톤 (초안 - /project-plan에서 확정)

- **M0 부트스트랩**: go 설치, `go run ./cmd/server` + `GET /healthz` 200. CI(lint/test).
- **M1 webhook 수신/정규화**: Grafana Alerting payload → 내부 Alert 구조체
  (cluster, namespace, app, rule UID/name, panel/dashboard URL). → 완료기준 1의 토대.
- **M2 enricher**: kubernetes(image tag/startTime) → argocd(repo/synced SHA) →
  github(커미터) → slack(lookupByEmail). 각 소스 인터페이스 + fake로 단위테스트. → 완료기준 4.
- **M3 notifier(발송)**: 메인 스레드 + 컨텍스트 댓글(패널/로그/rule). → 완료기준 1,3.
- **M4 silence 상호작용**: Slack 서명검증 → silence 모달 → Grafana Silences API. → 완료기준 2.
- **M5 ArgoCD rollback(플래그)**: rollback 버튼 + sync 트리거, `ARGOCD_ROLLBACK_ENABLED` 게이트. → 완료기준 5.
- **M6 배포**: Dockerfile(distroless) + k8s manifest(deploy/), Grafana contact point/Slack app 연동 문서.

## 리스크 / 미해결

- **이메일 매칭 불안정**: GitHub 커미터 이메일(noreply/사번) ≠ Slack 이메일인 경우.
  M2에서 fallback(커밋명 텍스트 표기) 필수. 필요 시 명시적 매핑 테이블 추가.
- **Go 미설치**: 현재 머신에 Go 없음. M0에서 설치 선행.
- **중앙 K8s 접근**: 멀티클러스터 pod 조회를 중앙 1개 kubeconfig로 할지, 클러스터별
  context를 cluster 라벨로 선택할지 M2 시작 시 확정.
