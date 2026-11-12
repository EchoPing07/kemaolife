package com.github.echoping07.kemaolife.ui

import android.os.Build
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.toggleable
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.data.isDarkTheme
import com.github.echoping07.kemaolife.data.normalizeThemeColor
import com.github.echoping07.kemaolife.ui.theme.AppThemeColor
import com.github.echoping07.kemaolife.ui.theme.previewColorScheme
import com.github.echoping07.kemaolife.ui.theme.smoothShape
import kotlin.math.min

/** 外观页只提供内容；滚动由 CollapsingScreen 的单一 verticalScroll 负责。 */
@Composable
internal fun AppearanceScreen(state: AppState, viewModel: AppViewModel) {
    val dark = isDarkTheme(state.themeMode, isSystemInDarkTheme())
    val dynamicAvailable = Build.VERSION.SDK_INT >= Build.VERSION_CODES.S

    // 颜色模式不在这里：它是设置总览「个性化」组的常驻分段控件（见 ThemeModeSelector）。
    Column(modifier = Modifier.fillMaxWidth()) {
        AppearanceSectionLabel(AppearanceText.DisplayAndColor, topPadding = 0.dp)
        GroupSurface(GroupPosition.FIRST, modifier = Modifier.fillMaxWidth()) {
            AppearanceSwitchRow(
                title = AppearanceText.DynamicColor,
                summary = when {
                    !dynamicAvailable -> AppearanceText.DynamicColorUnavailable
                    state.dynamicColor -> AppearanceText.DynamicColorEnabledSummary
                    else -> AppearanceText.DynamicColorDisabledSummary
                },
                checked = state.dynamicColor,
                enabled = dynamicAvailable,
                onCheckedChange = viewModel::setDynamicColor,
            )
        }
        GroupGap()
        GroupSurface(GroupPosition.LAST, modifier = Modifier.fillMaxWidth()) {
            AppearanceSwitchRow(
                title = AppearanceText.Amoled,
                summary = if (dark && state.amoled) AppearanceText.AmoledActiveSummary
                else AppearanceText.AmoledInactiveSummary,
                checked = state.amoled,
                // 偏好随时可设；纯黑背景只由 KemaolifeTheme 在深色下应用。
                enabled = true,
                onCheckedChange = viewModel::setAmoled,
            )
        }

        AppearanceSectionLabel(AppearanceText.PresetThemes)
        GroupSurface(GroupPosition.ONLY, modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(horizontal = 20.dp, vertical = 20.dp)) {
                Text(
                    text = AppearanceText.PresetThemesSummary,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(bottom = 18.dp),
                )
                AppThemeColor.entries.chunked(4).forEachIndexed { index, rowThemes ->
                    if (index > 0) Spacer(Modifier.height(16.dp))
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        rowThemes.forEach { theme ->
                            ThemeDonutWheel(
                                theme = theme,
                                selected = normalizeThemeColor(state.themeColor) == theme.name,
                                dark = dark,
                                modifier = Modifier.weight(1f),
                                onClick = { viewModel.setThemeColor(theme.name) },
                            )
                        }
                        repeat(4 - rowThemes.size) { Spacer(Modifier.weight(1f)) }
                    }
                }
            }
        }
    }
}

@Composable
private fun AppearanceSectionLabel(text: String, topPadding: androidx.compose.ui.unit.Dp = 22.dp) {
    Text(
        text = text,
        style = MaterialTheme.typography.labelLarge,
        fontWeight = FontWeight.SemiBold,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(top = topPadding, bottom = 8.dp),
    )
}

@Composable
private fun AppearanceSwitchRow(
    title: String,
    summary: String,
    checked: Boolean,
    enabled: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    Row(
        modifier = Modifier
            .toggleable(
                value = checked,
                enabled = enabled,
                role = Role.Switch,
                onValueChange = onCheckedChange,
            )
            .padding(start = 20.dp, end = 16.dp, top = 15.dp, bottom = 15.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f).padding(end = 12.dp)) {
            Text(text = title, style = MaterialTheme.typography.titleMedium)
            Text(
                text = summary,
                style = MaterialTheme.typography.bodySmall,
                color = if (enabled) MaterialTheme.colorScheme.onSurfaceVariant
                else MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f),
                modifier = Modifier.padding(top = 3.dp),
            )
        }
        Switch(checked = checked, onCheckedChange = null, enabled = enabled)
    }
}

/** 四个色阶环绕主色；选中勾位于主色中心，选中态不靠额外彩色底板。 */
@Composable
private fun ThemeDonutWheel(
    theme: AppThemeColor,
    selected: Boolean,
    dark: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val preview = theme.previewColorScheme(dark)
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = modifier
            .clip(smoothShape(PageCorners.Field))
            .selectable(selected = selected, onClick = onClick, role = Role.RadioButton)
            .padding(vertical = 8.dp),
    ) {
        Box(modifier = Modifier.size(68.dp), contentAlignment = Alignment.Center) {
            Canvas(modifier = Modifier.size(64.dp)) {
                val outerRadius = min(size.width, size.height) / 2f
                val innerRadius = outerRadius * 0.44f
                val strokeWidth = outerRadius - innerRadius
                val arcSize = Size(outerRadius * 2 - strokeWidth, outerRadius * 2 - strokeWidth)
                val topLeft = Offset(strokeWidth / 2f, strokeWidth / 2f)
                val colors = listOf(
                    preview.primary,
                    preview.primaryContainer,
                    preview.secondaryContainer,
                    preview.tertiaryContainer,
                )
                colors.forEachIndexed { index, color ->
                    drawArc(
                        color = color,
                        startAngle = -90f + index * 90f,
                        sweepAngle = 90f,
                        useCenter = false,
                        style = Stroke(width = strokeWidth),
                        size = arcSize,
                        topLeft = topLeft,
                    )
                }
                drawCircle(color = preview.primary, radius = innerRadius * 0.88f)
            }
            if (selected) {
                Icon(
                    imageVector = Icons.Rounded.Check,
                    contentDescription = null,
                    tint = if (dark) Color(0xFF17191C) else Color.White,
                    modifier = Modifier.size(18.dp),
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
