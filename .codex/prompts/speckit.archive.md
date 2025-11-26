---
description: 归档已完成的 speckit 规格目录（001-*）到 specs/archive，使用仓库内置脚本完成移动。
handoffs: []
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

1. 校验用户输入是否为 `001-*` 格式的规格目录名（例：`001-youtube-download`）。若为空或不匹配：ERROR 并提示需提供合法目录名。
2. 在仓库根运行归档脚本：  
   `.specify/scripts/bash/archive-spec.sh <feature-dir>`
   - 脚本会将 `specs/<feature-dir>` 移动到 `specs/archive/<feature-dir>`。
   - 若源目录不存在或目标已存在则报错。
3. 确认结果：输出归档后的新路径或脚本错误信息。

## Notes

- 回报时使用绝对路径。
- 不修改规格内容，只通过脚本移动目录。
- 如有多个目录名，按顺序逐个调用脚本归档。
