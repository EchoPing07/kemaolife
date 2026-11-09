package com.github.echoping07.kemaolife.ui

/**
 * 启动时序的纯策略（docs/002 §5.3「每次打开」）。
 *
 * 为什么单独抽出来：ViewModel 依赖 Android / JNI，JVM 单测跑不了，而「门面 Init
 * 慢于 ON_RESUME 时挂起、Init 完成后只补跑一次」是最容易回归的隐性契约。
 * ViewModel 只保留 `pendingOpen` 状态位，判定逻辑都在这里。
 */
internal object StartupTiming {

    /** Init 未就绪：这次打开必须挂起，不能撞 Bridge 的未初始化拦截。 */
    fun shouldDefer(ready: Boolean): Boolean = !ready

    /**
     * 是否消费挂起的打开：只有「Init 就绪」且「确实挂起过」才返回 true。
     * Init 仍失败时返回 false，`pendingOpen` 保留到下一次成功 Init（例如设置页保存）。
     */
    fun consumePending(ready: Boolean, pending: Boolean): Boolean = ready && pending
}
