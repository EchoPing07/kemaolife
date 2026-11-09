package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.ui.theme.AppThemeColor
import com.github.echoping07.kemaolife.ui.theme.palettePreview
import kotlin.math.min

/**
 * 外观设置：MD3 Expressive 四象限色轮盘选择器。
 * 遵循 Material Design 3 公开调色规范，实时换肤。
 */
@Composable
internal fun AppearanceScreen(state: AppState, viewModel: AppViewModel) {
    val dark = isSystemInDarkTheme()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
    ) {
        // 顶部说明横幅卡片
        Surface(
            color = MaterialTheme.colorScheme.surfaceContainerLow,
            shape = MaterialTheme.shapes.extraLarge,
            modifier = Modifier.fillMaxWidth().padding(top = 4.dp),
        ) {
            Row(
                modifier = Modifier.padding(18.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    modifier = Modifier
                        .size(40.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.primaryContainer),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = TabIcons.Appearance,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(20.dp),
                    )
                }
                Spacer(Modifier.width(14.dp))
                Column {
                    Text(
                        text = "Material 动态调色",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        text = stringResource(R.string.settings_theme_hint),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 2.dp),
                    )
                }
            }
        }

        Spacer(Modifier.height(16.dp))

        // 预设主题大圆角纯白卡片
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.surfaceContainerLow,
            ),
            shape = MaterialTheme.shapes.extraLarge,
        ) {
            Column(modifier = Modifier.padding(horizontal = 18.dp, vertical = 20.dp)) {
                Text(
                    text = "预设主题",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = "公开的 Material 3 基础色彩体系",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 2.dp, bottom = 18.dp),
                )

                // 每行排 3~4 个，使用弹性布局
                val chunks = AppThemeColor.entries.chunked(3)
                chunks.forEachIndexed { index, rowColors ->
                    if (index > 0) Spacer(Modifier.height(14.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceAround,
                    ) {
                        rowColors.forEach { theme ->
                            val selected = state.themeColor == theme.name
                            ThemeDonutWheel(
                                theme = theme,
                                selected = selected,
                                dark = dark,
                                onClick = { viewModel.setThemeColor(theme.name) },
                            )
                        }
                        // 补齐最后一行不足 3 个的占位
                        if (rowColors.size < 3) {
                            repeat(3 - rowColors.size) {
                                Spacer(Modifier.size(80.dp))
                            }
                        }
                    }
                }
            }
        }

        Spacer(Modifier.height(28.dp))
    }
}

/**
 * 四象限色轮盘：
 * 外圈展示 Primary, PrimaryContainer, SecondaryContainer, TertiaryContainer 4 个色阶切片，
 * 中心实心圆展示主色并支持打勾标记。
 */
@Composable
private fun ThemeDonutWheel(
    theme: AppThemeColor,
    selected: Boolean,
    dark: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val preview = theme.palettePreview(dark)
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = modifier
            .clip(MaterialTheme.shapes.large)
            .clickable(role = Role.RadioButton, onClick = onClick)
            .padding(vertical = 8.dp, horizontal = 6.dp),
    ) {
        Box(
            modifier = Modifier.size(64.dp),
            contentAlignment = Alignment.Center,
        ) {
            Canvas(modifier = Modifier.size(56.dp)) {
                val w = size.width
                val h = size.height
                val center = androidx.compose.ui.geometry.Offset(w / 2f, h / 2f)
                val outerRadius = min(w, h) / 2f
                val innerRadius = outerRadius * 0.44f
                val strokeWidth = outerRadius - innerRadius
                val arcSize = androidx.compose.ui.geometry.Size(
                    outerRadius * 2 - strokeWidth,
                    outerRadius * 2 - strokeWidth,
                )
                val topLeft = androidx.compose.ui.geometry.Offset(strokeWidth / 2f, strokeWidth / 2f)

                // 象限 1 (右上, -90° ~ 0°): primary
                drawArc(
                    color = preview.primary,
                    startAngle = -90f,
                    sweepAngle = 90f,
                    useCenter = false,
                    style = Stroke(width = strokeWidth),
                    size = arcSize,
                    topLeft = topLeft,
                )
                // 象限 2 (右下, 0° ~ 90°): primaryContainer
                drawArc(
                    color = preview.primaryContainer,
                    startAngle = 0f,
                    sweepAngle = 90f,
                    useCenter = false,
                    style = Stroke(width = strokeWidth),
                    size = arcSize,
                    topLeft = topLeft,
                )
                // 象限 3 (左下, 90° ~ 180°): secondaryContainer
                drawArc(
                    color = preview.secondaryContainer,
                    startAngle = 90f,
                    sweepAngle = 90f,
                    useCenter = false,
                    style = Stroke(width = strokeWidth),
                    size = arcSize,
                    topLeft = topLeft,
                )
                // 象限 4 (左上, 180° ~ 270°): tertiaryContainer
                drawArc(
                    color = preview.tertiaryContainer,
                    startAngle = 180f,
                    sweepAngle = 90f,
                    useCenter = false,
                    style = Stroke(width = strokeWidth),
                    size = arcSize,
                    topLeft = topLeft,
                )

                // 中心实心圆核
                drawCircle(
                    color = preview.primary,
                    radius = innerRadius * 0.88f,
                    center = center,
                )
            }

            // 选中标记
            if (selected) {
                Icon(
                    imageVector = Icons.Rounded.Check,
                    contentDescription = null,
                    tint = Color.White,
                    modifier = Modifier.size(16.dp),
                )
            }
        }

        Text(
            text = theme.label,
            style = MaterialTheme.typography.labelMedium,
            fontWeight = if (selected) FontWeight.Bold else FontWeight.Medium,
            color = if (selected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
            modifier = Modifier.padding(top = 8.dp),
        )
    }
}
