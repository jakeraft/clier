// ── Project Typography Hierarchy ──
//
// 전체 레벨 (위 → 아래):
//
//   Level 1 (brand)         : text-lg (18px) font-bold            icon size-7 (28px)  ← typography  ← navbar "Clier" logo
//   Level 2 (page title)    : text-lg (18px) font-semibold        icon size-6 (24px)  ← typography  ← page-level heading
//     DialogTitle / AlertDialogTitle                                                    ← shadcn      ← Level 2와 동급
//   Level 3 (section title) : text-sm (14px) font-semibold        icon size-5 (20px)  ← typography  ← SectionCard title
//     CardTitle                                                                        ← shadcn      ← Level 3와 동급
//   Level 4 (subsection)    : text-sm (14px) font-medium          icon size-4 (16px)  ← typography  ← table group header
//     Button (default/sm/lg): text-sm (14px) font-medium          icon size-4 (16px)  ← shadcn
//     Toggle                : text-sm (14px) font-medium          icon size-4 (16px)  ← shadcn
//     Label                 : text-sm (14px) font-medium                               ← shadcn
//   Level 5 (body)          : text-sm (14px)                                           ← typography  ← 일반 본문
//   Level 6 (muted)         : text-sm (14px) text-muted-foreground                     ← typography  ← 보조 텍스트, 날짜
//   Level 7 (caption)       : text-xs (12px) text-muted-foreground                     ← typography  ← 라벨, 타임스탬프
//     Badge                 : text-xs (12px) font-medium          icon size-3 (12px)  ← shadcn
//     Button (xs)           : text-xs (12px)                      icon size-3 (12px)  ← shadcn
//     Tooltip               : text-xs (12px)                                           ← shadcn
//   Level 8 (code)          : text-xs (12px) font-mono                                 ← typography  ← 코드/로그
//   ── icon-only levels ──
//   Level 5 (micro)         :                                    icon size-2.5 (10px)  ← typographyIcon ← graph node inline controls

export const typography = {
  1: "text-lg font-bold",
  2: "text-lg font-semibold",
  3: "text-sm font-semibold",
  4: "text-sm font-medium",
  5: "text-sm",
  6: "text-sm text-muted-foreground",
  7: "text-xs text-muted-foreground",
  8: "text-xs font-mono",
} as const;

export const typographyIcon = {
  1: "size-7",
  2: "size-6",
  3: "size-5",
  4: "size-4",
  5: "size-2.5",
} as const;
