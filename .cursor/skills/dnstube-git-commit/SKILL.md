---
name: dnstube-git-commit
description: Write concise conventional commits for DnsTube changes. Use when the user asks for a commit message or staged commit for this repository.
---

# DnsTube 提交说明

- 使用英文短句；前缀建议：`feat:`、`fix:`、`chore:`、`docs:`。
- 说明「做了什么」与涉及模块（如 `dns`、`api`、`web`），避免无关文件混入同一提交。

## 终止条件

出现以下任一情况时，必须停止执行 `git commit`，先说明并等待处理：

- 用户未明确要求“提交/commit”，仅要求修改代码或给提交信息建议。
- 发现疑似敏感信息（如 `.env`、密钥、令牌、凭证）被纳入暂存区。
- 预提交检查、测试或类型检查失败，且无法在当前上下文快速修复。
- 提交将包含自动生成或大体量噪声文件（构建产物、缓存、锁文件漂移）且用户未要求。
- 需要执行高风险 Git 操作（如 `--amend`、强推）但用户未明确授权。
- 单次提交出现多份可合并的数据库 `migrations sql` 文件
