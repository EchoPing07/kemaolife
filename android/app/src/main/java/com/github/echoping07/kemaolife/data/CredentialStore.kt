package com.github.echoping07.kemaolife.data

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import org.json.JSONObject
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * 凭据存储：Android Keystore（P5）。
 *
 * 做法：Keystore 里生成一把不可导出的 AES 密钥，用它以 AES/GCM 加密凭据 JSON，
 * 密文（IV + 数据）落到应用私有 SharedPreferences。明文永远不落盘、不进日志。
 *
 * 为什么不用 EncryptedSharedPreferences：那需要额外依赖（security-crypto 已废弃且
 * 需要新依赖，AGENTS A5 要求先确认）。这里用标准库 + Keystore 直接实现，能力等价：
 *  - 密钥材料在 TEE / StrongBox 内，进程拿不到；
 *  - 换机、清数据后密文无法解密，属于预期行为（不做云端同步）。
 */
class CredentialStore(private val context: Context) {

    private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    /**
     * 读凭据；没有存档或解密失败时返回默认值。
     *
     * 解密失败不抛给 UI：可能是换机 / 系统重置了 Keystore。此时按「未配置」处理，
     * 让用户重新填一次，比一个永远打不开的崩溃页好。
     */
    fun load(): Credentials {
        val stored = prefs.getString(KEY_CIPHERTEXT, null) ?: return Credentials()
        return try {
            val raw = Blob.decode(stored)
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.DECRYPT_MODE, secretKey(), GCMParameterSpec(GCM_TAG_BITS, raw.iv))
            val plain = cipher.doFinal(raw.data)
            fromJson(JSONObject(String(plain, Charsets.UTF_8)))
        } catch (e: Exception) {
            Credentials()
        }
    }

    /**
     * 写凭据。
     *
     * 空密码字段的语义是「不修改」：教务密码与校园网密码都是不可回显的，
     * 保存时只填了别的字段不应该把已存密码冲掉。用 [previous] 把旧值补回去。
     */
    fun save(next: Credentials, previous: Credentials): Credentials {
        val merged = next.copy(
            eduPassword = next.eduPassword.ifBlank { previous.eduPassword },
            campusPassword = next.campusPassword.ifBlank { previous.campusPassword },
        )
        persist(merged)
        return merged
    }

    private fun persist(credentials: Credentials) {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, secretKey())
        val data = cipher.doFinal(credentials.toJson().toString().toByteArray(Charsets.UTF_8))
        prefs.edit().putString(KEY_CIPHERTEXT, Blob(cipher.iv, data).encode()).apply()
    }

    private fun secretKey(): SecretKey {
        val ks = KeyStore.getInstance(ANDROID_KEYSTORE).apply { load(null) }
        (ks.getEntry(KEY_ALIAS, null) as? KeyStore.SecretKeyEntry)?.let { return it.secretKey }

        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, ANDROID_KEYSTORE)
        generator.init(
            KeyGenParameterSpec.Builder(
                KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                // 不设 setUserAuthenticationRequired：查询课表不应强制每次解锁。
                // 设备已解锁即可用，符合「打开即用」的产品定位。
                .build(),
        )
        return generator.generateKey()
    }

    private fun Credentials.toJson() = JSONObject()
        .put("studentId", studentId)
        .put("eduPassword", eduPassword)
        .put("customerCode", customerCode)
        .put("campusUser", campusUser)
        .put("campusPassword", campusPassword)
        .put("campusFingerprint", campusFingerprint)

    private fun fromJson(obj: JSONObject) = Credentials(
        studentId = obj.optString("studentId"),
        eduPassword = obj.optString("eduPassword"),
        customerCode = obj.optString("customerCode").ifBlank { Credentials.DEFAULT_CUSTOMER_CODE },
        campusUser = obj.optString("campusUser"),
        campusPassword = obj.optString("campusPassword"),
        campusFingerprint = obj.optString("campusFingerprint"),
    )

    /** 密文打包：IV 长度固定 12 字节，前缀拼接后整体 Base64。 */
    private class Blob(val iv: ByteArray, val data: ByteArray) {
        fun encode(): String {
            val out = ByteArray(1 + iv.size + data.size)
            out[0] = iv.size.toByte()
            iv.copyInto(out, 1)
            data.copyInto(out, 1 + iv.size)
            return Base64.encodeToString(out, Base64.NO_WRAP)
        }

        companion object {
            fun decode(s: String): Blob {
                val raw = Base64.decode(s, Base64.NO_WRAP)
                val ivLen = raw[0].toInt()
                require(ivLen in 12..16) { "非法 IV 长度" }
                return Blob(
                    iv = raw.copyOfRange(1, 1 + ivLen),
                    data = raw.copyOfRange(1 + ivLen, raw.size),
                )
            }
        }
    }

    private companion object {
        const val ANDROID_KEYSTORE = "AndroidKeyStore"
        const val KEY_ALIAS = "kemaolife.credentials"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val GCM_TAG_BITS = 128
        const val PREFS_NAME = "kemaolife.credentials"
        const val KEY_CIPHERTEXT = "ciphertext"
    }
}
