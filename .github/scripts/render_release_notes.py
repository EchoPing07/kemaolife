#!/usr/bin/env python3
"""渲染 Release 正文（docs/release.md §6）：更新内容 + 三张下载表。

必需环境变量：TAG / VERSION / ANDROID_CERT_SHA256；可选：PREV_TAG / RANGE / GITHUB_REPOSITORY。
存在 .github/release-notes/<TAG>.md 时，其内容替代自动分组的变更清单。
"""
import os
import re
import subprocess

TAG = os.environ["TAG"]
VERSION = os.environ["VERSION"]
CERT = os.environ["ANDROID_CERT_SHA256"]
PREV = os.environ.get("PREV_TAG", "")
RANGE = os.environ.get("RANGE") or (f"{PREV}..{TAG}" if PREV else TAG)
REPO = os.environ.get("GITHUB_REPOSITORY", "EchoPing07/kemaolife")
BASE = f"https://github.com/{REPO}"
DL = f"{BASE}/releases/download/{TAG}"

GROUPS = ("Mobile", "Desktop", "CLI", "Other")
SCOPE_GROUP = {
    "android": "Mobile",
    "desktop": "Desktop",
    "windows": "Desktop",
    "cli": "CLI",
    "skill": "CLI",
}


def auto_changelog():
    subjects = subprocess.run(
        ["git", "log", "--no-merges", "--reverse", "--pretty=%s", RANGE],
        capture_output=True, text=True, encoding="utf-8", check=True,
    ).stdout.splitlines()
    grouped = {g: [] for g in GROUPS}
    for subject in subjects:
        m = re.match(r"^\w+(?:\(([^)]*)\))?!?:\s*(.+)$", subject)
        scope, desc = (m.group(1) or "", m.group(2)) if m else ("", subject)
        # squash 合并可能带多个尾随 PR 号（如 (#24) (#25)），只保留最后一个作链接。
        prs = re.search(r"((?:\s*\(#\d+\))+)\s*$", desc)
        if prs:
            num = re.findall(r"\d+", prs.group(1))[-1]
            desc = desc[: prs.start()].rstrip() + f" ([#{num}]({BASE}/pull/{num}))"
        grouped[SCOPE_GROUP.get(scope, "Other")].append(f"- {desc}")
    sections = [f"### {g}\n\n" + "\n".join(items) for g, items in grouped.items() if items]
    return "\n\n".join(sections) if sections else "本版无代码变更。"


def row(system, arch, asset):
    return f"| {system} | {arch} | [{asset}]({DL}/{asset}) |"


def table(rows):
    return "| 系统 | 架构 | 下载 |\n|---|---|---|\n" + "\n".join(rows)


def main():
    override = f".github/release-notes/{TAG}.md"
    if os.path.exists(override):
        with open(override, encoding="utf-8") as f:
            changelog = f.read().strip()
        source = override
    else:
        changelog = auto_changelog()
        source = f"git log {RANGE}"

    v = VERSION
    cli = table([
        row("Linux", "x86_64", f"kemaolife_{v}_linux_amd64.tar.gz"),
        row("Linux", "arm64", f"kemaolife_{v}_linux_arm64.tar.gz"),
        row("Linux", "armv7（32 位）", f"kemaolife_{v}_linux_armv7.tar.gz"),
        row("macOS", "Apple Silicon（M1 及以后）", f"kemaolife_{v}_darwin_arm64.tar.gz"),
        row("macOS", "Intel", f"kemaolife_{v}_darwin_amd64.tar.gz"),
        row("跨平台", "agent skill（依赖同机 `kemaolife` 二进制）", f"kemaolife-skill_{v}.zip"),
    ])
    desktop = table([
        row("Windows", "x64", f"kemaolife-desktop_{v}_windows_amd64.zip"),
        row("Windows", "arm64（未实机验证）", f"kemaolife-desktop_{v}_windows_arm64.zip"),
    ])
    mobile = table([
        row("Android", "arm64-v8a（多数手机）", f"kemaolife_{v}_android_arm64-v8a.apk"),
        row("Android", "armeabi-v7a（旧 32 位机）", f"kemaolife_{v}_android_armeabi-v7a.apk"),
        row("Android", "x86（模拟器）", f"kemaolife_{v}_android_x86.apk"),
        row("Android", "x86_64（模拟器）", f"kemaolife_{v}_android_x86_64.apk"),
    ])

    parts = [
        "## 更新内容",
        changelog,
        "## 下载",
        "### CLI",
        cli,
        f"> macOS 包未经 Apple 签名与公证、未经实机测试，按试验性支持使用；"
        f"Gatekeeper 拦截处理见 [docs/cli.md §6]({BASE}/blob/{TAG}/docs/cli.md)。",
        "### Desktop",
        desktop,
        f"> 无代码签名：SmartScreen 提示时点「更多信息 → 仍要运行」，签名申请进展见 "
        f"[#4]({BASE}/issues/4)。解压后双击 `install.bat` 安装，详见 "
        f"[docs/windows.md]({BASE}/blob/{TAG}/docs/windows.md)。",
        "### Mobile",
        mobile,
        f"> 需 Android 7.0 及以上；升级直接覆盖安装同架构 APK，不要先卸载（会清掉凭据与课表快照）。"
        f"发布签名证书 SHA-256 指纹：`{CERT}`；从第三方渠道获取的 APK 请先用 "
        f"`apksigner verify --print-certs` 比对，不一致不要安装。安装说明见 "
        f"[docs/android.md]({BASE}/blob/{TAG}/docs/android.md)。",
        f"校验和：[SHA256SUMS]({DL}/SHA256SUMS)（以上全部包）",
    ]
    if PREV:
        parts.append(f"**Full Changelog**: {BASE}/compare/{PREV}...{TAG}")

    os.makedirs("dist", exist_ok=True)
    with open("dist/RELEASE_NOTES.md", "w", encoding="utf-8") as f:
        f.write("\n\n".join(parts) + "\n")
    print(f"dist/RELEASE_NOTES.md 已生成（变更来源：{source}）")


if __name__ == "__main__":
    main()