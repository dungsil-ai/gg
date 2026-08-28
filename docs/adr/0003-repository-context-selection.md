# 저장소 문맥 선택 계약

`gg`는 현재 Git remote를 자동 선택하는 forge 명령에 `--remote <name>`을 제공한다. `repo list`, `repo view`, `issue list/view/create`, `pr list/view/create`에서 쓸 수 있으며 `gg --remote upstream issue list`와 `gg issue list --remote upstream`은 같은 뜻이다.

`--repo`와 `--remote`를 함께 쓰면 어느 저장소를 고를지 숨기지 않고 usage 오류와 exit code `2`를 낸다. `repo create`는 새 URL을 뜻하는 `--repo`를 계속 요구하고, `clone`은 입력 URL을 쓰며, `pull`과 `push`는 남은 인자를 Git에 그대로 전달한다.

명시한 remote가 없으면 exit code `1`로 실패하고 사용할 수 있는 remote 이름을 보여준다. Flag가 없을 때는 branch upstream, `origin`, 유일 remote 순서의 기존 자동 선택을 유지한다.

`--repo`가 항상 이기는 방식과 `--remote`가 항상 이기는 방식도 검토했다. 잘못된 저장소에 Issue나 PR을 만들 수 있는 숨은 우선순위를 피하기 위해 둘을 함께 쓰지 못하게 했다.
