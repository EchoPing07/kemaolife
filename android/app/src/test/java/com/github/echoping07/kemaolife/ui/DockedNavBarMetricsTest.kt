package com.github.echoping07.kemaolife.ui

import androidx.compose.ui.unit.dp
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DockedNavBarMetricsTest {

    @Test
    fun contentHeightMatchesLabelVisibility() {
        assertEquals(64.dp, DockedNavBarMetrics.contentHeight(showLabels = true))
        assertEquals(56.dp, DockedNavBarMetrics.contentHeight(showLabels = false))
        assertTrue(
            DockedNavBarMetrics.contentHeight(showLabels = false) <
                DockedNavBarMetrics.contentHeight(showLabels = true),
        )
    }
}
