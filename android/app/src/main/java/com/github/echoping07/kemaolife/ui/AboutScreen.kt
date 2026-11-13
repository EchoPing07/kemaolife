package com.github.echoping07.kemaolife.ui

import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.widget.Toast
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.BuildConfig
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.ui.theme.appShape

/** 公开的项目地址，不拼任何本机数据（P1 / P2）。 */
private const val REPOSITORY_URL = "https://github.com/EchoPing07/kemaolife"
private const val LICENSE_URL = "$REPOSITORY_URL/blob/main/LICENSE"
private const val RELEASES_URL = "$REPOSITORY_URL/releases"

@Composable
internal fun AboutScreen(state: AppState, viewModel: AppViewModel) {
    val context = LocalContext.current
    fun open(url: String) = openExternalLink(context, url)
    Column(
        modifier = Modifier.fillMaxWidth(),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Surface(shape = appShape(PageCorners.Logo), modifier = Modifier.padding(top = 4.dp).size(104.dp)) {
            Image(
                painter = painterResource(R.mipmap.ic_about_logo),
                contentDescription = stringResource(R.string.app_name),
                contentScale = androidx.compose.ui.layout.ContentScale.Crop,
                modifier = Modifier.fillMaxSize(),
            )
        }
        Text(
            text = "KeMao Life",
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.SemiBold,
            modifier = Modifier.padding(top = 14.dp),
        )
        Text(
            text = stringResource(R.string.app_name),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 4.dp, bottom = 20.dp),
        )
        AboutRow(
            TabIcons.Version,
            stringResource(R.string.about_version),
            "${BuildConfig.VERSION_NAME} / ${BuildConfig.VERSION_CODE}",
            position = GroupPosition.FIRST,
        )
        GroupGap()
        AboutRow(
            icon = TabIcons.Device,
            title = stringResource(R.string.about_system),
            value = "${Build.MANUFACTURER} ${Build.MODEL}\nAndroid ${Build.VERSION.RELEASE} / SDK ${Build.VERSION.SDK_INT}",
            position = GroupPosition.LAST,
        )
        Spacer(Modifier.padding(top = 16.dp))
        AboutRow(
            icon = TabIcons.Github,
            title = "GitHub",
            value = "EchoPing07/kemaolife",
            onClick = { open(REPOSITORY_URL) },
            position = GroupPosition.FIRST,
        )
        GroupGap()
        AboutRow(
            icon = TabIcons.License,
            title = stringResource(R.string.about_license),
            value = stringResource(R.string.about_license_value),
            onClick = { open(LICENSE_URL) },
            position = GroupPosition.LAST,
        )
        Spacer(Modifier.padding(top = 16.dp))
        // 独立成组：上面两行是「跳外链」，这一行是「联网 → 出结果 → 弹半屏层」，行为不同。
        AboutRow(
            icon = TabIcons.Refresh,
            title = stringResource(R.string.about_check_update),
            value = stringResource(R.string.about_check_update_hint),
            onClick = viewModel::openUpdateCheck,
            position = GroupPosition.ONLY,
        )
    }

    state.updateCheck?.let { sheet ->
        UpdateCheckSheet(
            state = sheet,
            onDismiss = viewModel::closeUpdateCheck,
            onOpenReleases = { open(RELEASES_URL) },
        )
    }
}

/**
 * 三个外链共用一个入口：没有接收 ACTION_VIEW 的应用时提示一次，不让点击直接崩溃。
 * LocalContext 通常是 Activity，非 Activity 上下文仍需补 NEW_TASK。
 */
private fun openExternalLink(context: Context, url: String) {
    val intent = Intent(Intent.ACTION_VIEW, Uri.parse(url))
    if (context !is Activity) intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    try {
        context.startActivity(intent)
    } catch (e: ActivityNotFoundException) {
        Toast.makeText(context, R.string.open_link_failed, Toast.LENGTH_SHORT).show()
    }
}

@Composable
private fun AboutRow(
    icon: ImageVector,
    title: String,
    value: String,
    onClick: (() -> Unit)? = null,
    position: GroupPosition,
) {
    GroupSurface(position = position, onClick = onClick, modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(25.dp),
            )
            Spacer(Modifier.width(18.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(text = title, style = MaterialTheme.typography.titleMedium)
                Text(
                    text = value,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 3.dp),
                )
            }
        }
    }
}
