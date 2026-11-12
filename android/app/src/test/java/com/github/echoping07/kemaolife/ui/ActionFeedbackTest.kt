package com.github.echoping07.kemaolife.ui

import com.github.echoping07.kemaolife.data.ApiError
import com.github.echoping07.kemaolife.data.AppException
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ActionFeedbackTest {
    @Test
    fun `begin clears previous feedback`() {
        val controller = ActionFeedbackController()
        val first = controller.begin()
        assertTrue(controller.complete(first, ActionFeedback.SUCCESS))

        controller.begin()

        assertNull(controller.result)
    }

    @Test
    fun `stale request cannot overwrite current click`() {
        val controller = ActionFeedbackController()
        val first = controller.begin()
        val second = controller.begin()

        assertFalse(controller.complete(first, ActionFeedback.SUCCESS))
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
