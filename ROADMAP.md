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

### 기반
- [ ] **M0 toolchain + 스켈레톤 기동** - 검증: Go 설치 후 `go vet ./...` clean + `go run ./cmd/server` & `curl -s localhost:8080/healthz` → `ok`. (차단: Go 미설치)
- [ ] **M1 config 로더** - 검증: env → `Config` 단위테스트(필수값 누락 시 error, 플래그 파싱). (의존: M0)
- [ ] **M2 webhook 파싱/정규화** - 검증: 실제 Grafana Alerting webhook JSON 샘플 → `Alert{cluster,namespace,app,ruleUID,ruleName,panelURL}` 추출 table test. (의존: M0)

### enricher (소스별 독립 - M3~M6 병렬 가능, 모두 의존: M1)
- [ ] **M3 k8s enricher** - 검증: fake clientset에 pod 주입 → image tag + `startTime` + cluster 라벨 반환 단위테스트.
- [ ] **M4 argocd enricher** - 검증: `httptest` fake ArgoCD Application API → app명 → repo URL + synced revision(SHA) 반환.
- [ ] **M5 github enricher** - 검증: `httptest` fake GitHub `/commits/{sha}` → 커미터 name/email 반환.
- [ ] **M6 slack identity** - 검증: fake Slack `users.lookupByEmail` → email→userID; 미존재 email → fallback 마커(graceful) 반환.
- [ ] **M7 enrichment 파이프라인** - 검증: M3~M6 fake 전부 묶은 통합테스트 → `Alert`에 cluster/app/repo/deployer 채워짐. (의존: M2,M3,M4,M5,M6) → 완료기준 4

### notifier
- [ ] **M8 메시지 빌더(Block Kit)** - 검증: enriched `Alert` → 메인 블록(cluster/app/repo/요약) + 댓글 블록(패널·로그 링크·rule·@멘션) golden test(필드 assert). 순수 포맷, 네트워크 없음. → 완료기준 1,3 토대
- [ ] **M9 Slack 발송** - 검증: fake Slack가 메인 `chat.postMessage` → 동일 `thread_ts`로 댓글 수신; 또는 테스트 채널 수동발송 시 필드 포함 메시지 게시. (의존: M7,M8) → 완료기준 1,3

### silence 상호작용
- [ ] **M10 interactivity 엔드포인트 + 서명검증** - 검증: 서명된 payload POST → 200, 변조 서명 → 401 단위테스트. (의존: M1)
- [ ] **M11 silence 모달 → Grafana Silences API** - 검증: 모달 제출 핸들러 → fake Grafana Silences에 POST 후 GET으로 silence 존재 확인. (의존: M10) → 완료기준 2

### ArgoCD 액션 (플래그)
- [ ] **M12 rollback 버튼 + sync 트리거** - 검증: 플래그 off → 빌드된 블록에 버튼 없음(assert); on → 클릭 핸들러가 fake ArgoCD에 직전 revision sync 호출. (의존: M4,M8,M10) → 완료기준 5

### 배포
- [ ] **M13 컨테이너화** - 검증: distroless `Dockerfile` `docker build` 성공 + `docker run` 후 `curl /healthz` 200. (의존: M0)
- [ ] **M14 k8s 매니페스트 + 연동 문서** - 검증: `kubectl apply --dry-run=client` 통과 + Grafana contact point/Slack app 연동 체크리스트 문서화. (의존: M9,M11)

## 후순위 / 보류
- **명시적 GitHub↔Slack 매핑 테이블** - 보류: lookupByEmail + fallback으로 시작. 이메일 불일치가 실제로 빈번해질 때 도입(추측성 선구현 금지).
- **멀티 Grafana/분산 토폴로지** - 보류: 중앙 집중형 결정. 실제 분산 요구 시 재논의.
- **멀티클러스터 K8s context 선택 로직** - 보류: 단일 kubeconfig로 시작, 실제 멀티 pod 조회 필요 시 M3 확장.
- **rollback 외 추가 액션(재시작 등)** - 보류: 요청 범위 밖.

## 외부 차단 요소 (작업 전 확보 필요)
- **Go 설치** - M0 차단.
- **자격증명**: Grafana/ArgoCD/GitHub/Slack 토큰 + 테스트 Slack 채널 + 도달 가능한 중앙 Grafana - M9/M11/M13~14의 live 검증 차단(M3~M8 fake 단위테스트는 불요).
- **Slack App 생성**: interactivity Request URL은 공개 인그레스 필요 - M10/M11 live 검증 차단.

## 완료 기록
- (아직 없음) M0 스켈레톤은 커밋(`52fd7fd`)됐으나 Go 미설치로 `go run` 검증 미완 → M0 open 유지.
