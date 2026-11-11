package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.ui.theme.smoothShape

/** MD3 Expressive 组内条目位置：3dp 缝隙显露页面底色，不画分隔线。 */
internal enum class GroupPosition { ONLY, FIRST, MIDDLE, LAST }

@Composable
internal fun GroupSurface(
    position: GroupPosition,
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    val large = 26.dp
    val small = 7.dp
    val shape = when (position) {
        GroupPosition.ONLY -> smoothShape(large)
        GroupPosition.FIRST -> smoothShape(large, small)
        GroupPosition.MIDDLE -> smoothShape(small)
        GroupPosition.LAST -> smoothShape(small, large)
    }
    if (onClick == null) {
        Surface(
            modifier = modifier,
            shape = shape,
            color = MaterialTheme.colorScheme.surfaceContainerLow,
            content = { Column(content = content) },
        )
    } else {
        Surface(
            modifier = modifier,
            shape = shape,
            color = MaterialTheme.colorScheme.surfaceContainerLow,
            onClick = onClick,
            content = { Column(content = content) },
        )
    }
}

@Composable
internal fun GroupGap() {
    Spacer(Modifier.height(3.dp))
}
