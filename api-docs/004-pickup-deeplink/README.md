# 取件码深链（Pickup Code Deep Links）

> **状态**：整理自开源项目 [pickup-code-launcher](https://github.com/EidenLiu/pickup-code-launcher)（`cn.pickup.launcher`）的公开实现，未在本仓库逐条实机验证
> **更新**：2026-11-16
> **适用**：Android 跳转第三方 App 的取件码 / 待取快递页；本仓库只采用两个取件码入口（见 §6）

**这不是查询协议：全部动作只是 `ACTION_VIEW` 隐式 Intent 跳转，不发任何网络请求、不携带凭据、不需要后端。链接均为各 App 公开注册的 scheme 与官方 H5 地址，属公开分发事实，不含个人隐私（P1）。**

---

## 1. 核心事实

| 项 | 值 |
|---|---|
| 跳转方式 | `Intent(ACTION_VIEW, uri)`；私有 scheme 候选**锁定目标包名**（`setPackage`），避免弹选择器或被浏览器截走 |
| 回退链 | 私有 scheme 候选（按序、钉包名）→ 官方 H5 钉包名 → 官方 H5 交系统浏览器 → 本机提示（参考实现另有「仅菜鸟拉起 App 首页」特例，本仓库不采用，见 §2①） |
| 成败判定 | 只能判断「有没有接收者」：`startActivity` 抛 `ActivityNotFoundException` / `SecurityException` 等即失败，继续下一条；目标 App 收到后内部失败这里看不到 |
| Intent flags | `FLAG_ACTIVITY_CLEAR_TOP \| FLAG_ACTIVITY_SINGLE_TOP` 复用已有页面栈；非 Activity 上下文补 `FLAG_ACTIVITY_NEW_TASK` |
| 包可见性 | Android 11+ `startActivity` 不受包可见性限制；`getLaunchIntentForPackage` / `resolveActivity` 需要 Manifest `<queries>` 声明目标包 |
| 权限 | 不需要网络权限，不读短信 / 通知 / 相册 / 账号数据 |

---

## 2. 目标链接

每个目标 = 包名 + 一组**按优先级排列**的私有 scheme 候选 + 一个官方 H5 兜底地址。候选从「参数最全」到「裸路由」逐级降级，兼容不同版本 App 对参数的接受程度。

### ① 菜鸟身份码（`com.cainiao.wireless`，本仓库不采用，仅存档）

```
cainiao://desktop/station_code
```

H5 兜底：

```
https://market.m.taobao.com/app/cn-yz/multi-activity/authCode.html?bizEntry=ALIPAY_GUOGUO
```

**特例**：菜鸟没有公开直达链接时，先用 `getLaunchIntentForPackage` 拉起官方 App 首页并本机提示「请在 App 内查看取件码」，再走 H5 回退（参考实现的 `openAppWhenDeepLinkUnavailable=true`，七个目标中仅此一个）。

### ② 淘宝取件码（`com.taobao.taobao`）

```
taobao://m.taobao.com/tbopen/index.html?h5Url=https%3A%2F%2Fpages-fast.m.taobao.com%2Fwow%2Fz%2Funiapp%2F1011717%2Flast-mile-fe%2Fend-collect-platform%2Fidentity-code%3Fx-ssr%3Dtrue
tbopen://m.taobao.com/tbopen/index.html?h5Url=https%3A%2F%2Fpages-fast.m.taobao.com%2Fwow%2Fz%2Funiapp%2F1011717%2Flast-mile-fe%2Fend-collect-platform%2Fidentity-code%3Fx-ssr%3Dtrue
```

H5 兜底：

```
https://pages-fast.m.taobao.com/wow/z/uniapp/1011717/last-mile-fe/end-collect-platform/identity-code?x-ssr=true
```

`tbopen/index.html?h5Url=<percent-encoded url>` 是淘宝的统一路由壳：内层 H5 地址整体 percent-encode 一次，`taobao://` 与 `tbopen://` 两个 scheme 对应不同版本的注册。

### ③ 拼多多身份码（`com.xunmeng.pinduoduo`）

```
pinduoduo://com.xunmeng.pinduoduo/mdkd/package?tab=ID_CODE&entry_source=11&refer_page_name=login&refer_page_sn=10169
pinduoduo://com.xunmeng.pinduoduo/mdkd/package?tab=ID_CODE&entry_source=11
pinduoduo://com.xunmeng.pinduoduo/mdkd/package
```

H5 兜底：

```
https://m.pinduoduo.net/mdkd/package?entry_source=18&extra_params=from_wx_jump%3D1&p_channel=0
```

| 参数 | 含义 |
|---|---|
| `mdkd/package` | App 内快递 / 取件页路由 |
| `tab=ID_CODE` | 落在「身份码」标签页；不带则停在默认标签（待取列表） |
| `entry_source` / `refer_page_name` / `refer_page_sn` | 模仿 App 内部跳转携带的来源参数，非鉴权字段；旧版本可能不认，故候选逐级去参 |

H5 兜底是微信跳转用的公开入口（`extra_params=from_wx_jump%3D1`）；部分版本拼多多自己注册了该域名的 App Links，钉包名发送可在 App 内渲染。

### ④ 待取快递入口（本仓库不采用，仅存档）

| 目标 | 包名 | 私有 scheme | H5 兜底 |
|---|---|---|---|
| 淘宝待取快递 | `com.taobao.taobao` | `taobao://` / `tbopen://m.taobao.com/tbopen/index.html?h5Url=https%3A%2F%2Fpages-fast.m.taobao.com%2Fwow%2Fz%2Funiapp%2F1100333%2Flast-mile-fe%2Fm-end-school-tab%2Fhome` | `https://pages-fast.m.taobao.com/wow/z/uniapp/1100333/last-mile-fe/m-end-school-tab/home` |
| 拼多多待取快递 | `com.xunmeng.pinduoduo` | `pinduoduo://com.xunmeng.pinduoduo/mdkd/package` | 同 ③ |
| 京东待取快递 | `com.jingdong.app.mall` | `openjd://virtual?params=%7B%22category%22%3A%22jump%22%2C%22des%22%3A%22orderlist%22%7D`（内层是 percent-encoded JSON `{"category":"jump","des":"orderlist"}`） | `https://order.jd.com/center/list.action` |
| 小红书待取快递 | `com.xingin.xhs` | `xhsdiscover://rn/lancer-order/order/list` | `https://www.xiaohongshu.com/` |

---

## 3. 回退顺序（逐条对照实现）

1. **私有 scheme 候选按序尝试**：每条都 `setPackage(目标包名)`；任一条有接收者且启动成功即结束。
2. **H5 钉包名**：部分版本官方 App 注册了自己的 HTTPS App Links，能直接在 App 内渲染该页，优先于浏览器。
3. **H5 不钉包名**：交系统浏览器 / 用户默认接收者。
4. **全部失败**：只给本机提示，不崩溃、不静默。

参考实现在第 1、2 步之间还有「（仅菜鸟）拉起 App 首页」特例：`getLaunchIntentForPackage` 非空则启动并本机提示「没有公开直达链接，已打开官方 App」，未安装或无启动入口则继续。本仓库不采用菜鸟入口，该特例不实现。

注意：

- 失败原因只记「无接收者 / 系统拒绝」类别，不把上游文案带进提示。
- 隐式 Intent 无法感知目标 App 内部失败（如 H5 要求登录），提示只在整条链都没有接收者时给出。
- 参考实现用 `resolveActivity` 预判接收者；等价做法是直接 `startActivity` + 捕获异常（本仓库采用后者，与「交流」入口同一模式，少一次包可见性查询）。

---

## 4. 隐私与边界

| 项 | 要求 |
|---|---|
| 链接内容 | 只含公开 scheme / 官方 H5 地址与固定来源参数，**不拼任何本机数据**（学号 / 快照 / 凭据 / 日志 / 房间号） |
| 数据读取 | 不读短信、通知、相册、账号数据；不代查他人，取件码内容只在目标 App 内展示 |
| 网络 | 跳转本身 0 请求；H5 兜底由浏览器 / 目标 App 自行加载，与本仓库查询链路无关 |
| 服务化 | 不做对外服务或批量入口（P4 同旨）；仅本人设备手动点击触发 |

---

## 5. 待验证

| 项 | 说明 |
|---|---|
| 各链接在当前 App 版本的可用性 | 整理自参考项目公开实现，未逐条实机验证；App 改版可能使候选失效，回退链兜底 |
| H5 兜底的登录要求 | 浏览器打开官方 H5 可能要求登录 / 无法展示身份码，属目标 App 行为，不在本链路控制 |
| 菜鸟直达链接 | `cainiao://desktop/station_code` 在部分版本无接收者，参考实现故有「拉起 App 首页」特例；本仓库不采用菜鸟入口 |

---

## 6. 本仓库采用范围

Android「更多」页 `取件码` 卡片只采用 **② 淘宝取件码 / ③ 拼多多身份码** 两个入口（完整回退链）；① 菜鸟身份码与 ④ 待取快递入口不实现，仅存档。实现位于 Kotlin 呈现层（`android/.../ui/PickupCode.kt`），不进 `core` / 门面、不新增 `op`；功能行为见 [docs/android.md §4](../../docs/android.md#4-tab2-更多)。

---

## 7. 变更记录

| 日期 | 内容 |
|---|---|
| 2026-11-16 | 建档：整理 pickup-code-launcher 公开实现的七个目标深链、回退顺序、Intent flags 与包可见性口径；声明本仓库只采用三个取件码入口，待取快递入口仅存档 |
| 2026-11-16 | 采用范围收敛为 ② 淘宝 / ③ 拼多多两个入口：① 菜鸟身份码改为仅存档，「拉起 App 首页」特例不再实现；§1 回退链、§3 回退顺序与 §5 / §6 同步 |