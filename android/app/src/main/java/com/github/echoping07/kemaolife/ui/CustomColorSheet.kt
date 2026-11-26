package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.data.formatThemeColorHex
import com.github.echoping07.kemaolife.data.normalizeCustomThemeColor
import com.github.echoping07.kemaolife.data.parseThemeColorHex
import com.github.echoping07.kemaolife.ui.theme.CustomThemePalette
import com.github.echoping07.kemaolife.ui.theme.appShape
import com.github.echoping07.kemaolife.ui.theme.customColorScheme

private val QuickPickSeedColors = listOf(
    0xFF26A69A.toInt(), // 青绿
    0xFF2563EB.toInt(), // 钴蓝
    0xFF4F46E5.toInt(), // 靛青
    0xFF9333EA.toInt(), // 紫罗兰
    0xFFEC4899.toInt(), // 樱粉
    0xFFE11D48.toInt(), // 绯红
    0xFFD97706.toInt(), // 琥珀
    0xFF15803D.toInt(), // 松绿
)

private val HueSpectrumStops = listOf(
    Color(0xFFFF0000),
    Color(0xFFFFFF00),
    Color(0xFF00FF00),
    Color(0xFF00FFFF),
    Color(0xFF0000FF),
    Color(0xFFFF00FF),
    Color(0xFFFF0000),
)

/**
 * 外观页「自定义主题色」底部选色弹层：
 * 支持 HSV 二维明度/饱和度色板、色相滑条、推荐色点与 Hex 输入，并实时预览派生出的 M3 色阶。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun CustomColorSheet(
    initialColor: Int,
    dark: Boolean,
    amoled: Boolean,
    dynamicColorActive: Boolean,
    onDismiss: () -> Unit,
    onApply: (Int) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val focusManager = LocalFocusManager.current
    val initialNormalized = remember(initialColor) { normalizeCustomThemeColor(initialColor) }
    val initialHsv = remember(initialNormalized) { CustomThemePalette.rgbToHsv(initialNormalized) }

    // 独立保存 HSV 分量：拖到纯白 / 纯黑时不会丢失当前色相，避免色相滑条跳回红色。
    var hue by remember(initialNormalized) { mutableFloatStateOf(initialHsv.hue) }
    var saturation by remember(initialNormalized) { mutableFloatStateOf(initialHsv.saturation) }
    var value by remember(initialNormalized) { mutableFloatStateOf(initialHsv.value) }
    var selectedArgb by remember(initialNormalized) { mutableIntStateOf(initialNormalized) }
    var hexText by remember(initialNormalized) { mutableStateOf(formatThemeColorHex(initialNormalized)) }
    var hexInvalid by remember(initialNormalized) { mutableStateOf(false) }

    fun syncFromHsv(nextHue: Float, nextSat: Float, nextVal: Float) {
        hue = nextHue.coerceIn(0f, 360f)
        saturation = nextSat.coerceIn(0f, 1f)
        value = nextVal.coerceIn(0f, 1f)
        val argb = CustomThemePalette.hsvToRgb(hue, saturation, value)
        selectedArgb = argb
        hexText = formatThemeColorHex(argb)
        hexInvalid = false
    }

    fun applySeedColor(argb: Int) {
        val normalized = normalizeCustomThemeColor(argb)
        val hsv = CustomThemePalette.rgbToHsv(normalized)
        if (hsv.saturation > 0f) {
            hue = hsv.hue
        }
        saturation = hsv.saturation
        value = hsv.value
        selectedArgb = normalized
        hexText = formatThemeColorHex(normalized)
        hexInvalid = false
    }

    val previewScheme = remember(selectedArgb, dark, amoled) {
        customColorScheme(selectedArgb, dark = dark, amoled = amoled)
    }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        shape = appShape(PageCorners.Card),
        containerColor = MaterialTheme.colorScheme.surfaceContainerLow,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .imePadding()
                .navigationBarsPadding()
                .padding(horizontal = 20.dp),
        ) {
            Text(
                text = AppearanceText.CustomThemeSheetTitle,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.SemiBold,
            )
            Spacer(Modifier.height(14.dp))

            // 实时配色预览：展示当前种子色在当前深浅色（含 AMOLED）下派生的主色与容器色。
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(appShape(PageCorners.Nested))
                    .background(previewScheme.surface)
                    .border(
                        width = 1.dp,
                        color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f),
                        shape = appShape(PageCorners.Nested),
                    )
                    .padding(horizontal = 16.dp, vertical = 14.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                ThemePaletteDonut(
                    scheme = previewScheme,
                    selected = false,
                    dark = dark,
                    size = 52.dp,
                )
                Spacer(Modifier.width(14.dp))
                Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        PreviewRoleChip(
                            label = AppearanceText.CustomThemePreviewPrimary,
                            container = previewScheme.primary,
                            content = previewScheme.onPrimary,
                            modifier = Modifier.weight(1f),
                        )
                        PreviewRoleChip(
                            label = AppearanceText.CustomThemePreviewContainer,
                            container = previewScheme.primaryContainer,
                            content = previewScheme.onPrimaryContainer,
                            modifier = Modifier.weight(1f),
                        )
                    }
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        PreviewRoleChip(
                            label = AppearanceText.CustomThemePreviewSecondary,
                            container = previewScheme.secondaryContainer,
                            content = previewScheme.onSecondaryContainer,
                            modifier = Modifier.weight(1f),
                        )
                        PreviewRoleChip(
                            label = AppearanceText.CustomThemePreviewTertiary,
                            container = previewScheme.tertiaryContainer,
                            content = previewScheme.onTertiaryContainer,
                            modifier = Modifier.weight(1f),
                        )
                    }
                }
            }

            Spacer(Modifier.height(16.dp))

            // HSV 二维色板（横轴饱和度，纵轴明度）。
            SaturationValuePanel(
                hue = hue,
                saturation = saturation,
                value = value,
                onChange = { nextSat, nextVal ->
                    focusManager.clearFocus()
                    syncFromHsv(hue, nextSat, nextVal)
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(176.dp),
            )

            Spacer(Modifier.height(14.dp))

            // 360° 色相光谱条。
            HueSliderBar(
                hue = hue,
                onHueChange = { nextHue ->
                    focusManager.clearFocus()
                    syncFromHsv(nextHue, saturation, value)
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(28.dp),
            )

            Spacer(Modifier.height(16.dp))

            // 推荐色快捷点选。
            Text(
                text = AppearanceText.CustomThemeQuickPicks,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(10.dp))
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                QuickPickSeedColors.forEach { swatch ->
                    val swatchSelected = normalizeCustomThemeColor(selectedArgb) == swatch
                    val checkTint = if (CustomThemePalette.relativeLuminance(swatch) > 0.45) {
                        Color(0xFF17191C)
                    } else {
                        Color.White
                    }
                    Box(
                        modifier = Modifier
                            .size(34.dp)
                            .clip(CircleShape)
                            .background(Color(swatch))
                            .border(
                                width = if (swatchSelected) 2.dp else 1.dp,
                                color = if (swatchSelected) {
                                    MaterialTheme.colorScheme.onSurface
                                } else {
                                    MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f)
                                },
                                shape = CircleShape,
                            )
                            .clickable(role = Role.RadioButton) {
                                focusManager.clearFocus()
                                applySeedColor(swatch)
                            },
                        contentAlignment = Alignment.Center,
                    ) {
                        if (swatchSelected) {
                            Icon(
                                imageVector = Icons.Rounded.Check,
                                contentDescription = null,
                                tint = checkTint,
                                modifier = Modifier.size(16.dp),
                            )
                        }
                    }
                }
            }

            Spacer(Modifier.height(16.dp))

            // Hex 精确输入：与上方色板双向同步。
            OutlinedTextField(
                value = hexText,
                onValueChange = { raw ->
                    val filtered = raw.filter { it == '#' || it in '0'..'9' || it in 'a'..'f' || it in 'A'..'F' }
                        .take(7)
                    hexText = filtered
                    val parsed = parseThemeColorHex(filtered)
                    if (parsed != null) {
                        val hsv = CustomThemePalette.rgbToHsv(parsed)
                        if (hsv.saturation > 0f) {
                            hue = hsv.hue
                        }
                        saturation = hsv.saturation
                        value = hsv.value
                        selectedArgb = parsed
                        hexInvalid = false
                    } else {
                        hexInvalid = true
                    }
                },
                label = { Text(AppearanceText.CustomThemeHexLabel) },
                placeholder = { Text(AppearanceText.CustomThemeHexPlaceholder) },
                singleLine = true,
                isError = hexInvalid,
                supportingText = if (hexInvalid) {
                    { Text(AppearanceText.CustomThemeHexError) }
                } else {
                    null
                },
                leadingIcon = {
                    Box(
                        modifier = Modifier
                            .size(22.dp)
                            .clip(CircleShape)
                            .background(Color(selectedArgb))
                            .border(
                                width = 1.dp,
                                color = MaterialTheme.colorScheme.outlineVariant,
                                shape = CircleShape,
                            ),
                    )
                },
                shape = appShape(PageCorners.Field),
                keyboardOptions = KeyboardOptions(
                    capitalization = KeyboardCapitalization.Characters,
                    imeAction = ImeAction.Done,
                ),
                keyboardActions = KeyboardActions(
                    onDone = {
                        val parsed = parseThemeColorHex(hexText)
                        if (parsed != null) {
                            applySeedColor(parsed)
                        }
                        focusManager.clearFocus()
                    },
                ),
                modifier = Modifier.fillMaxWidth(),
            )

            if (dynamicColorActive) {
                Spacer(Modifier.height(8.dp))
                Text(
                    text = AppearanceText.CustomThemeDynamicNotice,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }

            Spacer(Modifier.height(18.dp))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                OutlinedButton(
                    onClick = onDismiss,
                    shape = CircleShape,
                    modifier = Modifier
                        .weight(1f)
                        .height(48.dp),
                ) {
                    Text(
                        text = AppearanceText.CustomThemeCancel,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
                Button(
                    onClick = { onApply(selectedArgb) },
                    enabled = !hexInvalid,
                    shape = CircleShape,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = previewScheme.primary,
                        contentColor = previewScheme.onPrimary,
                    ),
                    modifier = Modifier
                        .weight(1f)
                        .height(48.dp),
                ) {
                    Text(
                        text = AppearanceText.CustomThemeApply,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
            Spacer(Modifier.height(20.dp))
        }
    }
}

@Composable
private fun PreviewRoleChip(
    label: String,
    container: Color,
    content: Color,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier = modifier
            .height(30.dp)
            .clip(CircleShape)
            .background(container),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            fontWeight = FontWeight.SemiBold,
            color = content,
        )
    }
}

@Composable
private fun SaturationValuePanel(
    hue: Float,
    saturation: Float,
    value: Float,
    onChange: (Float, Float) -> Unit,
    modifier: Modifier = Modifier,
) {
    val pureHueColor = remember(hue) { Color(CustomThemePalette.hsvToRgb(hue, 1f, 1f)) }
    val shape = appShape(PageCorners.Nested)
    val latestOnChange by rememberUpdatedState(onChange)

    fun update(position: Offset, size: IntSize) {
        if (size.width <= 0 || size.height <= 0) return
        val nextSat = (position.x / size.width.toFloat()).coerceIn(0f, 1f)
        val nextVal = (1f - position.y / size.height.toFloat()).coerceIn(0f, 1f)
        latestOnChange(nextSat, nextVal)
    }

    Canvas(
        modifier = modifier
            .clip(shape)
            .border(
                width = 1.dp,
                color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.45f),
                shape = shape,
            )
            .pointerInput(Unit) {
                awaitEachGesture {
                    // 消费按下与拖拽事件，避免底部弹层把面板内的纵向选色拖拽误判为下拉关闭。
                    val down = awaitFirstDown(requireUnconsumed = false)
                    down.consume()
                    update(down.position, size)
                    do {
                        val event = awaitPointerEvent()
                        val change = event.changes.firstOrNull { it.id == down.id } ?: break
                        if (change.pressed) {
                            change.consume()
                            update(change.position, size)
                        }
                    } while (event.changes.any { it.pressed })
                }
            },
    ) {
        drawRect(
            brush = Brush.horizontalGradient(listOf(Color.White, pureHueColor)),
        )
        drawRect(
            brush = Brush.verticalGradient(listOf(Color.Transparent, Color.Black)),
        )
        val cx = saturation.coerceIn(0f, 1f) * size.width
        val cy = (1f - value.coerceIn(0f, 1f)) * size.height
        val indicatorRadius = 11.dp.toPx()
        drawCircle(
            color = Color.Black.copy(alpha = 0.35f),
            radius = indicatorRadius + 1.5.dp.toPx(),
            center = Offset(cx, cy),
            style = Stroke(width = 2.dp.toPx()),
        )
        drawCircle(
            color = Color.White,
            radius = indicatorRadius,
            center = Offset(cx, cy),
            style = Stroke(width = 2.5.dp.toPx()),
        )
    }
}

@Composable
private fun HueSliderBar(
    hue: Float,
    onHueChange: (Float) -> Unit,
    modifier: Modifier = Modifier,
) {
    val thumbColor = remember(hue) { Color(CustomThemePalette.hsvToRgb(hue, 1f, 1f)) }
    val latestOnHueChange by rememberUpdatedState(onHueChange)

    fun update(position: Offset, size: IntSize) {
        if (size.width <= 0) return
        val nextHue = ((position.x / size.width.toFloat()).coerceIn(0f, 1f)) * 360f
        latestOnHueChange(nextHue)
    }

    Canvas(
        modifier = modifier
            .pointerInput(Unit) {
                awaitEachGesture {
                    val down = awaitFirstDown(requireUnconsumed = false)
                    down.consume()
                    update(down.position, size)
                    do {
                        val event = awaitPointerEvent()
                        val change = event.changes.firstOrNull { it.id == down.id } ?: break
                        if (change.pressed) {
                            change.consume()
                            update(change.position, size)
                        }
                    } while (event.changes.any { it.pressed })
                }
            },
    ) {
        val trackHeight = 14.dp.toPx()
        val trackTop = (size.height - trackHeight) / 2f
        val corner = CornerRadius(trackHeight / 2f, trackHeight / 2f)
        drawRoundRect(
            brush = Brush.horizontalGradient(HueSpectrumStops),
            topLeft = Offset(0f, trackTop),
            size = androidx.compose.ui.geometry.Size(size.width, trackHeight),
            cornerRadius = corner,
        )

        val thumbRadius = 12.dp.toPx()
        val usableWidth = (size.width - thumbRadius * 2f).coerceAtLeast(1f)
        val cx = thumbRadius + (hue.coerceIn(0f, 360f) / 360f) * usableWidth
        val cy = size.height / 2f
        drawCircle(
            color = Color.Black.copy(alpha = 0.28f),
            radius = thumbRadius + 1.dp.toPx(),
            center = Offset(cx, cy),
        )
        drawCircle(
            color = Color.White,
            radius = thumbRadius,
            center = Offset(cx, cy),
        )
        drawCircle(
            color = thumbColor,
            radius = thumbRadius - 3.dp.toPx(),
            center = Offset(cx, cy),
        )
    }
}
