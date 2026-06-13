# 로드맵 - Monitoring

> Goal: 중앙 Grafana Alerting이 firing한 알람을 받아 알람 시점에 K8s/ArgoCD/GitHub/Slack를
> 라이브 조회해 enrich하고, Slack 스레드(메인 + silence/rollback 상호작용)로 발송해
> 대상 클러스터/서비스/배포 책임자를 즉시 인지시킨다.
> 갱신: 2026-06-13

## 완료 기준 (Goal 달성 판정 - 검증 가능)

1. **E2E 발송**: 테스트 알람 → Slack 메인 스레드에 cluster + app/service(k8s) + 배포 repo name 포함.
2. **Silence 모달**: 스레드 Silence 버튼 → Block Kit 모달 제출 → Grafana Silences API에 생성(GET으로 확인).
3. **컨텍스트 댓글**: 댓글에 패널 링크 + 로그 원천(Kibana/Prometheus) 링크 + alert rule 이름.
4. **배포 책임자 태깅**: ArgoCD synced revision(SHA) 커미터 이메일 → `users.lookupByEmail` → @멘션(실패 시 graceful fallback).
5. **rollback 플래그**: rollback 버튼은 `ARGOCD_ROLLBACK_ENABLED=true`일 때만 동작, 직전 revision으로 sync.

설계 결정/리스크 배경은 [CLAUDE.md](CLAUDE.md), 아키텍처는 [README.md](README.md) 참조.

## 마일스톤

각 단계는 외부(K8s/Grafana/GitHub/Slack/ArgoCD)를 인터페이스로 추상화하고 fake로
단위검증한다. 네트워크 호출을 단위테스트에 넣지 않는다.

전 마일스톤 구현 완료. 코드/단위검증은 완료(`go test ./...` green), live 검증(실
자격증명/도구 필요)은 아래 "남은 live 검증"에 분리. 상세는 ## 완료 기록 참조.

### 기반
- [x] **M0 toolchain + 스켈레톤 기동**
- [x] **M1 config 로더** (+ 중앙 설정 파일 `.env.example`)
- [x] **M2 webhook 파싱/정규화**

### enricher (소스별 독립)
- [x] **M3 k8s enricher**
- [x] **M4 argocd enricher**
- [x] **M5 github enricher**
- [x] **M6 slack identity**
- [x] **M7 enrichment 파이프라인** → 완료기준 4

### notifier
- [x] **M8 메시지 빌더(Block Kit)** → 완료기준 1,3
- [x] **M9 Slack 발송** (unit) → 완료기준 1,3 / live는 아래

### silence 상호작용
- [x] **M10 interactivity 엔드포인트 + 서명검증**
- [x] **M11 silence 모달 → Grafana Silences API** (unit) → 완료기준 2 / live는 아래

### ArgoCD 액션 (플래그)
- [x] **M12 rollback 버튼 + sync 트리거** → 완료기준 5

### 배포
- [x] **M13 컨테이너화** (Dockerfile 작성) - `docker build`는 아래 live 검증
- [x] **M14 k8s 매니페스트 + 연동 문서** - `kubectl --dry-run`은 아래 live 검증

## 남은 live 검증 (자격증명/도구 확보 후)
- **M9/M11 실발송·실silence**: 실 Slack App + Grafana 토큰으로 테스트 채널 발송 및
  Silences API 생성 확인. (단위테스트로 로직은 검증됨)
- **M13 `docker build`**: 현재 머신 docker 미연동(WSL). 이미지 빌드/런 확인 필요.
- **M14 `kubectl apply --dry-run=client`**: 대상 클러스터에서 매니페스트 검증.

## 후순위 / 보류
- **명시적 GitHub↔Slack 매핑 테이블** - 보류: lookupByEmail + fallback으로 시작. 이메일 불일치가 실제로 빈번해질 때 도입(추측성 선구현 금지).
- **멀티 Grafana/분산 토폴로지** - 보류: 중앙 집중형 결정. 실제 분산 요구 시 재논의.
- **멀티클러스터 K8s context 선택 로직** - 보류: 단일 kubeconfig로 시작, 실제 멀티 pod 조회 필요 시 M3 확장.
- **rollback 외 추가 액션(재시작 등)** - 보류: 요청 범위 밖.

## 완료 기록
- [x] **M0~M14 구현** (2026-06-13) - 검증: `go build ./...`/`go vet ./...` clean,
  `go test ./...` 전 패키지 green, blank config로 서버 기동 후 `GET /healthz`→`ok`,
  `POST /webhook/grafana`(빈 alerts)→200, 서명 없는 `/slack/interactivity`→401 스모크 확인.
- 설계: 외부 의존성 0(stdlib만), 모든 연동 `{BaseURL,Token}` REST + 인터페이스+fake.
- Go go1.26 `/usr/local/go` 설치(개발 머신).
- 완료기준 1~5는 단위/스모크로 검증됨. 실 자격증명 기반 E2E는 "남은 live 검증" 참조.
