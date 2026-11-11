package com.github.echoping07.kemaolife.ui

import android.os.Build
import androidx.compose.foundation.background
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.toggleable
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.data.THEME_MODE_DARK
import com.github.echoping07.kemaolife.data.THEME_MODE_LIGHT
import com.github.echoping07.kemaolife.data.THEME_MODE_SYSTEM
import com.github.echoping07.kemaolife.data.isDarkTheme
import com.github.echoping07.kemaolife.data.normalizeThemeColor
import com.github.echoping07.kemaolife.ui.theme.AppThemeColor
import com.github.echoping07.kemaolife.ui.theme.previewColorScheme

/** 滚动与大标题由 SettingsScreen 的 CollapsingScreen 统一提供。 */
@Composable
internal fun AppearanceScreen(state: AppState, viewModel: AppViewModel) {
    val dark = isDarkTheme(state.themeMode, isSystemInDarkTheme())
    val dynamicAvailable = Build.VERSION.SDK_INT >= Build.VERSION_CODES.S

    AppearanceSectionLabel(AppearanceText.ColorMode, topPadding = 4.dp)
    GroupSurface(GroupPosition.ONLY, modifier = Modifier.fillMaxWidth()) {
        SingleChoiceSegmentedButtonRow(
            modifier = Modifier
                .fillMaxWidth()
                .padding(8.dp),
        ) {
            val modes = listOf(
                THEME_MODE_SYSTEM to AppearanceText.ModeSystem,
                THEME_MODE_LIGHT to AppearanceText.ModeLight,
                THEME_MODE_DARK to AppearanceText.ModeDark,
            )
            modes.forEachIndexed { index, (mode, label) ->
                SegmentedButton(
                    selected = state.themeMode == mode,
                    onClick = { viewModel.setThemeMode(mode) },
                    shape = SegmentedButtonDefaults.itemShape(index = index, count = modes.size),
                    modifier = Modifier.weight(1f),
                    label = { Text(label, maxLines = 1) },
                    icon = {},
                )
            }
        }
    }

    AppearanceSectionLabel(AppearanceText.DisplayAndColor)
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
            enabled = dark,
            onCheckedChange = viewModel::setAmoled,
        )
    }

    AppearanceSectionLabel(AppearanceText.Preview)
    AppearancePreview(
        theme = state.resolvedTheme(),
        dark = dark,
        amoled = state.amoled,
        dynamic = state.dynamicColor && dynamicAvailable,
    )

    AppearanceSectionLabel(AppearanceText.PresetThemes)
    Text(
        text = if (state.dynamicColor && dynamicAvailable) AppearanceText.PresetThemesDynamicSummary
        else AppearanceText.PresetThemesSummary,
        style = MaterialTheme.typography.bodySmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(horizontal = 12.dp, vertical = 4.dp),
    )
    Spacer(Modifier.height(8.dp))
    AppThemeColor.entries.forEachIndexed { index, theme ->
        if (index > 0) Spacer(Modifier.height(10.dp))
        PresetThemeCard(
            theme = theme,
            selected = state.resolvedTheme() == theme,
            dark = dark,
            amoled = state.amoled,
            onClick = { viewModel.setThemeColor(theme.name) },
        )
    }
    Spacer(Modifier.height(28.dp))
}

private fun AppState.resolvedTheme(): AppThemeColor =
    AppThemeColor.entries.firstOrNull { it.name == normalizeThemeColor(themeColor) } ?: AppThemeColor.BLUE

@Composable
private fun AppearanceSectionLabel(text: String, topPadding: androidx.compose.ui.unit.Dp = 20.dp) {
    Text(
        text = text,
        style = MaterialTheme.typography.titleSmall,
        fontWeight = FontWeight.SemiBold,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(start = 12.dp, end = 12.dp, top = topPadding, bottom = 8.dp),
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
            .padding(start = 18.dp, end = 12.dp, top = 15.dp, bottom = 15.dp),
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

@Composable
private fun AppearancePreview(theme: AppThemeColor, dark: Boolean, amoled: Boolean, dynamic: Boolean) {
    val scheme = if (dynamic) MaterialTheme.colorScheme else theme.previewColorScheme(dark, amoled)
    Surface(
        shape = MaterialTheme.shapes.extraLarge,
        color = scheme.surfaceContainer,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Surface(shape = MaterialTheme.shapes.large, color = scheme.surfaceContainerLow) {
                Row(modifier = Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = AppearanceText.PreviewBackground,
                            style = MaterialTheme.typography.titleMedium,
                            color = scheme.onSurface,
                        )
                        Text(
                            text = if (dynamic) AppearanceText.SystemPalette else theme.label,
                            style = MaterialTheme.typography.bodySmall,
                            color = scheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 3.dp),
                        )
                    }
                    Surface(shape = MaterialTheme.shapes.medium, color = scheme.primary) {
                        Text(
                            text = AppearanceText.PreviewAction,
                            style = MaterialTheme.typography.labelLarge,
                            color = scheme.onPrimary,
                            modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp),
                        )
                    }
                }
            }
            Row(modifier = Modifier.padding(top = 10.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                listOf(
                    scheme.primaryContainer to scheme.onPrimaryContainer,
                    scheme.secondaryContainer to scheme.onSecondaryContainer,
                    scheme.tertiaryContainer to scheme.onTertiaryContainer,
                ).forEach { (container, content) ->
                    Box(
                        modifier = Modifier
                            .weight(1f)
                            .height(36.dp)
                            .background(container, MaterialTheme.shapes.medium),
                        contentAlignment = Alignment.Center,
                    ) {
                        Box(Modifier.size(10.dp).background(content, MaterialTheme.shapes.small))
                    }
                }
            }
        }
    }
}

@Composable
private fun PresetThemeCard(
    theme: AppThemeColor,
    selected: Boolean,
    dark: Boolean,
    amoled: Boolean,
    onClick: () -> Unit,
) {
    val scheme = theme.previewColorScheme(dark, amoled)
    val container by animateColorAsState(
        if (selected) scheme.secondaryContainer else scheme.surfaceContainerLow,
        label = "themeSelection",
    )
    val checkScale by animateFloatAsState(
        if (selected) 1f else 0f,
        animationSpec = spring(dampingRatio = 0.65f),
        label = "themeCheck",
    )
    GroupSurface(
        position = GroupPosition.ONLY,
        onClick = onClick,
        modifier = Modifier
            .fillMaxWidth()
            .selectable(
                selected = selected,
                onClick = onClick,
                role = Role.RadioButton,
            ),
    ) {
        Row(
            modifier = Modifier.background(container).padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            PaletteStrip(scheme = scheme, modifier = Modifier.width(76.dp).height(64.dp))
            Text(
                text = theme.label,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Medium,
                color = if (selected) scheme.onSecondaryContainer else scheme.onSurface,
                modifier = Modifier.weight(1f).padding(horizontal = 16.dp),
            )
            Icon(
                imageVector = Icons.Rounded.Check,
                contentDescription = null,
                tint = scheme.onSecondaryContainer,
                modifier = Modifier.size(24.dp).graphicsLayer {
                    scaleX = checkScale
                    scaleY = checkScale
                    alpha = checkScale
                },
            )
        }
    }
}

@Composable
private fun PaletteStrip(scheme: androidx.compose.material3.ColorScheme, modifier: Modifier = Modifier) {
    val shape: Shape = MaterialTheme.shapes.medium
    Row(modifier = modifier.clip(shape).background(scheme.primaryContainer)) {
        Box(Modifier.width(24.dp).fillMaxHeight().background(scheme.primary))
        Column(modifier = Modifier.fillMaxSize().padding(start = 4.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Box(Modifier.fillMaxWidth().weight(1f).background(scheme.primaryContainer))
            Row(Modifier.fillMaxWidth().weight(1f), horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                Box(Modifier.weight(1f).fillMaxHeight().background(scheme.secondaryContainer))
                Box(Modifier.weight(1f).fillMaxHeight().background(scheme.tertiaryContainer))
            }
        }
    }
}
