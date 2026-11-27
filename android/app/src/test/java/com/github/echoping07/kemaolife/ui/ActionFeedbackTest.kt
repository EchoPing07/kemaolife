package com.github.echoping07.kemaolife.ui

import com.github.echoping07.kemaolife.data.ApiError
import com.github.echoping07.kemaolife.data.AppException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ActionFeedbackTest {
    private fun withController(
        clearDelayMs: Long = 10_000L,
        block: suspend CoroutineScope.(ActionFeedbackController) -> Unit,
    ) = runBlocking {
        val job = Job()
        val scope = CoroutineScope(coroutineContext + job)
        try {
            block(ActionFeedbackController(scope, clearDelayMs = clearDelayMs))
        } finally {
            job.cancel()
        }
    }

    @Test
    fun `begin clears previous feedback`() = withController { controller ->
        val first = controller.begin()
        assertTrue(controller.complete(first, ActionFeedback.SUCCESS))
        assertEquals(ActionFeedback.SUCCESS, controller.result)

        controller.begin()

        assertNull(controller.result)
    }

    @Test
    fun `stale request cannot overwrite current click`() = withController { controller ->
        val first = controller.begin()
        val second = controller.begin()

        assertFalse(controller.complete(first, ActionFeedback.SUCCESS))
        assertTrue(controller.complete(second, ActionFeedback.FAILURE))
        assertEquals(ActionFeedback.FAILURE, controller.result)
    }

    @Test
    fun `show displays immediate feedback and invalidates earlier click`() = withController { controller ->
        val earlier = controller.begin()

        controller.show(ActionFeedback.UNAVAILABLE)

        assertEquals(ActionFeedback.UNAVAILABLE, controller.result)
        assertFalse(controller.complete(earlier, ActionFeedback.SUCCESS))
        assertEquals(ActionFeedback.UNAVAILABLE, controller.result)
    }

    @Test
    fun `clear resets result and invalidates in-flight click`() = withController { controller ->
        val inFlight = controller.begin()
        controller.show(ActionFeedback.SUCCESS)

        controller.clear()

        assertNull(controller.result)
        assertFalse(controller.complete(inFlight, ActionFeedback.FAILURE))
        assertNull(controller.result)
    }

    @Test
    fun `complete auto-clears after delay`() = withController(clearDelayMs = 25L) { controller ->
        val id = controller.begin()
        assertTrue(controller.complete(id, ActionFeedback.SUCCESS))
        assertEquals(ActionFeedback.SUCCESS, controller.result)

        withTimeout(2_000L) {
            while (controller.result != null) delay(10L)
        }
        assertNull(controller.result)
    }

    @Test
    fun `show auto-clears after delay`() = withController(clearDelayMs = 25L) { controller ->
        controller.show(ActionFeedback.UNAVAILABLE)
        assertEquals(ActionFeedback.UNAVAILABLE, controller.result)

        withTimeout(2_000L) {
            while (controller.result != null) delay(10L)
        }
        assertNull(controller.result)
    }

    @Test
    fun `new begin cancels earlier auto-clear timer`() = withController(clearDelayMs = 30L) { controller ->
        val first = controller.begin()
        assertTrue(controller.complete(first, ActionFeedback.SUCCESS))

        val second = controller.begin()
        // 等到第一次的定时器窗口过去后，再完成第二次点击，不应被旧定时器误清。
        delay(60L)
        assertNull(controller.result)
        assertTrue(controller.complete(second, ActionFeedback.FAILURE))
        assertEquals(ActionFeedback.FAILURE, controller.result)
    }

    @Test
    fun `credential and session errors are unavailable`() {
        for (code in 3001..3003) {
            assertEquals(
                ActionFeedback.UNAVAILABLE,
                actionFeedbackFor(AppException(ApiError(code, "错误"))),
            )
        }
    }

    @Test
    fun `network and business errors are failures`() {
        for (code in listOf(1001, 2001, 4001, 4002, 5001)) {
            assertEquals(
                ActionFeedback.FAILURE,
                actionFeedbackFor(AppException(ApiError(code, "错误"))),
            )
        }
    }
}
