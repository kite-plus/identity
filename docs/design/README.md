# Kite Plus 账号 设计文档

`id.kite.plus` 是 Kite Plus 所有服务共用的登录，是基础设施。这些文档是**开发期的约束来源**：与文档冲突的实现，先改文档，再改代码。

各应用怎么接入（OIDC 流程、`(iss, sub)`、本地会话、退出、账号删除）是跨项目约定，写在 [Explore identity-and-comments.md](https://github.com/kite-plus/explore/blob/main/docs/design/identity-and-comments.md)；这里只写身份服务本身。

## 文档索引

| 文档 | 内容 | 读者 |
|---|---|---|
| [architecture.md](architecture.md) | 定位、各服务的要求、登录方式、选型、部署、运维、仓库结构 | 所有贡献者，**先读这篇** |
| [roadmap.md](roadmap.md) | I0–I3 各阶段的范围和退出条件，待定事项 | 所有贡献者，排期前读 |
| [../runbooks](../runbooks/README.md) | 首次部署、升级、恢复、密钥轮换、故障处理 | 运维 |

## 文档约定

沿用 [Kite 设计文档的约定](https://github.com/kite-plus/kite/blob/main/docs/design/README.md)：正文中文，专有名词保留英文；代码、注释、commit message 一律英文；`[待定]` 表示明确推迟决策；同一条规则只写在一个地方。身份服务的升级时限、备份和部署只写在本仓库，其他项目链接过来。

## 当前状态

| 文档 | 状态 | 最近更新 |
|---|---|---|
| architecture.md | 已选 ZITADEL（S1、S7 待 I1 确认）；R1–R11 的落实见 §1.1；部署在与 Explore 同机的服务器上（§4） | 2026-10-11 |
| roadmap.md | I0 完成；I1 本地部分完成，等 DNS 和账号 | 2026-10-11 |

2026-10-10 补充：手机应用经 Explore 服务端登录，身份服务里不登记原生客户端；为 App Store 加上 Apple 登录（R11、S7）。

2026-10-11：I0 实测完成，选定 ZITADEL v4.19.4；本地开发栈、初始化工具、部署文件和运维手册落地，运维手册在 [runbooks](../runbooks/README.md)。

2026-10-11：部署地改为与 Explore 同机的美国 RackNerd 服务器，Explore 的 Caddy 做整台机器的入口，身份服务以边缘模式接在共用的 `kite-edge` 网络上（architecture.md §4）；本机演练过。
