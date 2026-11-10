package com.github.echoping07.kemaolife.ui

/**
 * 区分「注册 observer 时补发的 ON_RESUME」与「真正从后台回到前台」。
 *
 * Lifecycle.addObserver 会把新 observer 同步推进到当前状态；冷启动注册时
 * Activity 已经是 RESUMED，因此会收到一次没有先经历 ON_STOP 的 ON_RESUME。
 * 把它当作「打开 App」会在门面 Init 完成前多发一轮自动查询（冷启动那一轮由
 * ViewModel 初始化链路在 Init 成功后补跑）。真正从后台返回若仍早于 Init，
 * 由 ViewModel 的 `pendingOpen` 挂起并在 Init 成功后消费，不会丢失。
 */
internal class AppOpenGate {
    private var stopped = false

    fun onStop() {
        stopped = true
    }

    /** 返回 true 表示这是一次新的前台打开，且只消费一次。 */
    fun onResume(): Boolean {
        if (!stopped) return false
        stopped = false
        return true
    }
}
