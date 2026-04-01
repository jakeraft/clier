// ── Project Layout System ──
//
// gap 스케일 (작은 → 큰):
//
//   0 : gap-0   ← flush 배치 (dense 버튼 그룹)
//   1 : gap-1   ← 요소 내부 (icon+text, badge, switch+label)
//   3 : gap-3   ← 요소 사이 (컨트롤 그룹 간 간격)
//   5 : gap-5   ← 영역 사이 (카드 섹션, 폼 섹션)
//
// flex 조합:
//
//   row     : flex items-center                  ← 수평 정렬 (기본)
//   col     : flex flex-col                      ← 수직 스택
//   colFill : flex flex-1 flex-col               ← 수직 스택 + 남은 공간 채움
//   between : flex items-center justify-between  ← 양쪽 정렬 (헤더, 타이틀↔액션)
//   center  : flex items-center justify-center   ← 완전 중앙 (empty state)
//   wrap    : flex flex-wrap                     ← 줄바꿈 (badge/tag 목록)
//   end     : flex justify-end                   ← 우측 정렬 (채팅 버블)
//   start   : flex justify-start                 ← 좌측 정렬 (채팅 버블)

export const gap = {
  0: "gap-0",
  1: "gap-1",
  3: "gap-3",
  4: "gap-4",
  5: "gap-5",
} as const;

export const flex = {
  row: "flex items-center",
  col: "flex flex-col",
  colFill: "flex flex-1 flex-col",
  between: "flex items-center justify-between",
  center: "flex items-center justify-center",
  wrap: "flex flex-wrap",
  end: "flex justify-end",
  start: "flex justify-start",
} as const;
