package com.github.echoping07.kemaolife.ui

import android.content.Intent
import android.net.Uri
import android.os.Build
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.BuildConfig
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.ui.theme.smoothShape

@Composable
internal fun AboutScreen() {
    val context = LocalContext.current
    fun open(url: String) = context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 20.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Surface(shape = smoothShape(32.dp), modifier = Modifier.padding(top = 16.dp).size(128.dp)) {
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
            modifier = Modifier.padding(top = 18.dp),
        )
        Text(
            text = stringResource(R.string.app_name),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 4.dp, bottom = 24.dp),
        )
        AboutGroup {
            AboutRow(TabIcons.Version, stringResource(R.string.about_version), "${BuildConfig.VERSION_NAME} / ${BuildConfig.VERSION_CODE}")
            Spacer(Modifier.padding(top = 2.dp))
            AboutRow(
                icon = TabIcons.Device,
                title = stringResource(R.string.about_system),
                value = "${Build.MANUFACTURER} ${Build.MODEL}\nAndroid ${Build.VERSION.RELEASE} / SDK ${Build.VERSION.SDK_INT}",
                position = AboutItemPosition.LAST,
            )
        }
        Spacer(Modifier.padding(top = 16.dp))
        AboutGroup {
            AboutRow(
                icon = TabIcons.Github,
                title = "GitHub",
                value = "EchoPing07/kemaolife",
                onClick = { open("https://github.com/EchoPing07/kemaolife") },
            )
            Spacer(Modifier.padding(top = 2.dp))
            AboutRow(
                icon = TabIcons.License,
                title = stringResource(R.string.about_license),
                value = stringResource(R.string.about_license_value),
                onClick = { open("https://github.com/EchoPing07/kemaolife/blob/main/LICENSE") },
                position = AboutItemPosition.LAST,
            )
        }
    }
}

private enum class AboutItemPosition { FIRST, LAST }

@Composable
private fun AboutGroup(content: @Composable () -> Unit) {
    Column(modifier = Modifier.fillMaxWidth()) { content() }
}

@Composable
private fun AboutRow(
    icon: ImageVector,
    title: String,
    value: String,
    onClick: (() -> Unit)? = null,
    position: AboutItemPosition = AboutItemPosition.FIRST,
) {
    val shape = if (position == AboutItemPosition.FIRST) smoothShape(28.dp, 4.dp)
    else smoothShape(4.dp, 28.dp)
    Surface(
        shape = shape,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        onClick = { onClick?.invoke() },
        enabled = onClick != null,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
                modifier = Modifier
                    .size(36.dp)
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.secondaryContainer)
                    .padding(8.dp),
            )
            Spacer(Modifier.width(16.dp))
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
