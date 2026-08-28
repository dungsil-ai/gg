# Domain Docs

이 저장소는 Single Context 구조를 쓴다.

## 코드 탐색 전에 읽기

- root `CONTEXT.md`가 있으면 관련 용어를 읽는다.
- `docs/adr/`가 있으면 작업 영역과 관련된 ADR을 읽는다.
- 파일이 없으면 조용히 계속한다. 미리 빈 파일을 만들지 않는다.

## 구조

```text
/
├── CONTEXT.md
├── docs/
│   └── adr/
│       ├── 0001-example.md
│       └── 0002-example.md
└── *.go
```

## 용어 규칙

- 출력, Issue 제목, test 이름에는 `CONTEXT.md`의 용어를 그대로 쓴다.
- 같은 뜻의 다른 말을 새로 만들지 않는다.
- 새 용어가 실제로 필요할 때만 `/vibe-modeling`으로 기록한다.

## ADR 규칙

- 기존 ADR과 다른 계획은 숨기지 않고 충돌을 밝힌다.
- 되돌리기 어렵고, 코드만 봐서는 이유를 알 수 없고, 실제 대안을 비교한 결정만 ADR로 남긴다.
