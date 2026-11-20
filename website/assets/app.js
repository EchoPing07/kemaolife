const RELEASE_BASE = 'https://github.com/EchoPing07/kemaolife/releases/download/v0.6.1/';

const OS_CONFIG = {
  android: {
    name: 'Android',
    navTarget: 'mobile',
    navLabel: '探索 Mobile 端特性',
    hint: '适用于 Android 7.0 及以上设备，支持无缝覆盖更新',
    packages: [
      {
        edition: 'Mobile · 移动端',
        arch: 'arm64-v8a', label: 'arm64-v8a（推荐）',
        file: 'kemaolife_0.6.1_android_arm64-v8a.apk',
        btnText: 'Mobile 移动端'
      },
      {
        edition: 'Mobile · 移动端',
        arch: 'armeabi-v7a', label: 'armeabi-v7a（32位）',
        file: 'kemaolife_0.6.1_android_armeabi-v7a.apk',
        btnText: 'Mobile 移动端 (32位)'
      },
      {
        edition: 'Mobile · 移动端',
        arch: 'x86_64', label: 'x86_64（模拟器）',
        file: 'kemaolife_0.6.1_android_x86_64.apk',
        btnText: 'Mobile 模拟器 (x86_64)'
      },
      {
        edition: 'Mobile · 移动端',
        arch: 'x86', label: 'x86（模拟器）',
        file: 'kemaolife_0.6.1_android_x86.apk',
        btnText: 'Mobile 模拟器 (x86)'
      }
    ]
  },
  windows: {
    name: 'Windows',
    navTarget: 'desktop',
    navLabel: '探索 Desktop 端特性',
    hint: '解压后运行 install.bat 即可完成部署并常驻系统托盘',
    packages: [
      {
        edition: 'Desktop · 桌面端',
        arch: 'amd64', label: 'amd64（推荐）',
        file: 'kemaolife-desktop_0.6.1_windows_amd64.zip',
        btnText: 'Desktop 桌面端'
      },
      {
        edition: 'Desktop · 桌面端',
        arch: 'arm64', label: 'arm64（ARM）',
        file: 'kemaolife-desktop_0.6.1_windows_arm64.zip',
        btnText: 'Desktop 桌面端 (ARM64)'
      }
    ]
  },
  macos: {
    name: 'MacOS',
    navTarget: 'cli',
    navLabel: '探索 CLI 端特性',
    hint: '单文件原生构建，Apple Silicon 机型请选择 arm64 版本',
    packages: [
      {
        edition: 'CLI · 命令行端',
        arch: 'arm64', label: 'arm64（M系列芯片）',
        file: 'kemaolife_0.6.1_darwin_arm64.tar.gz',
        btnText: 'CLI 命令行端'
      },
      {
        edition: 'CLI · 命令行端',
        arch: 'amd64', label: 'amd64（Intel）',
        file: 'kemaolife_0.6.1_darwin_amd64.tar.gz',
        btnText: 'CLI 命令行端'
      },
      {
        edition: 'CLI · AI Skill 扩展',
        arch: 'Skill', label: 'Agent Skill 包',
        file: 'kemaolife-skill_0.6.1.zip',
        btnText: 'CLI 技能包'
      }
    ]
  },
  linux: {
    name: 'Linux',
    navTarget: 'cli',
    navLabel: '探索 CLI 端特性',
    hint: '零外部依赖静态二进制，支持终端直查与 AI Agent 接入',
    packages: [
      {
        edition: 'CLI · 命令行端',
        arch: 'amd64', label: 'amd64（推荐）',
        file: 'kemaolife_0.6.1_linux_amd64.tar.gz',
        btnText: 'CLI 命令行端'
      },
      {
        edition: 'CLI · 命令行端',
        arch: 'arm64', label: 'arm64',
        file: 'kemaolife_0.6.1_linux_arm64.tar.gz',
        btnText: 'CLI 命令行端'
      },
      {
        edition: 'CLI · 命令行端',
        arch: 'armv7', label: 'armv7',
        file: 'kemaolife_0.6.1_linux_armv7.tar.gz',
        btnText: 'CLI 命令行端'
      },
      {
        edition: 'CLI · AI Skill 扩展',
        arch: 'Skill', label: 'Agent Skill 包',
        file: 'kemaolife-skill_0.6.1.zip',
        btnText: 'CLI 技能包'
      }
    ]
  }
};

const state = {
  activeView: 'home',
  selectedOS: 'android',
  selectedPkgIndex: 0
};

function detectOSFromUA() {
  const ua = navigator.userAgent.toLowerCase();
  if (ua.includes('android')) return 'android';
  if (ua.includes('win')) return 'windows';
  if (ua.includes('mac')) return 'macos';
  if (ua.includes('linux')) return 'linux';
  return 'android';
}

function matchPkgIndexByArch(os, archHint) {
  const pkgs = OS_CONFIG[os].packages;
  if (!archHint) return 0;
  const idx = pkgs.findIndex(p => p.arch && p.arch.toLowerCase() === archHint.toLowerCase());
  return idx >= 0 ? idx : 0;
}

function detectArchHeuristic(os) {
  const ua = navigator.userAgent.toLowerCase();
  const plat = (navigator.platform || '').toLowerCase();
  const combined = ua + ' ' + plat;

  if (os === 'android') {
    if (/x86_64|x64|amd64/.test(combined)) return 'x86_64';
    if (/i686|i386|x86/.test(combined)) return 'x86';
    if (/armv7/.test(combined)) return 'armeabi-v7a';
    return 'arm64-v8a';
  }
  if (os === 'windows') {
    if (/arm64|aarch64/.test(combined)) return 'arm64';
    return 'amd64';
  }
  if (os === 'linux') {
    if (/aarch64|arm64/.test(combined)) return 'arm64';
    if (/armv7/.test(combined)) return 'armv7';
    return 'amd64';
  }
  if (os === 'macos') {
    try {
      const canvas = document.createElement('canvas');
      const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
      if (gl) {
        const dbg = gl.getExtension('WEBGL_debug_renderer_info');
        if (dbg) {
          const renderer = (gl.getParameter(dbg.UNMASKED_RENDERER_WEBGL) || '').toLowerCase();
          if (renderer.includes('intel') || renderer.includes('amd radeon')) return 'amd64';
          if (renderer.includes('apple')) return 'arm64';
        }
      }
    } catch (e) {}
    return 'arm64';
  }
  return '';
}

async function autoDetectEnvironment() {
  let os = detectOSFromUA();
  let archHint = detectArchHeuristic(os);

  if (navigator.userAgentData && navigator.userAgentData.getHighEntropyValues) {
    try {
      const info = await navigator.userAgentData.getHighEntropyValues(['architecture', 'bitness', 'platform']);
      const plat = (info.platform || '').toLowerCase();
      if (plat.includes('android')) os = 'android';
      else if (plat.includes('win')) os = 'windows';
      else if (plat.includes('mac')) os = 'macos';
      else if (plat.includes('linux')) os = 'linux';

      const rawArch = (info.architecture || '').toLowerCase();
      const bits = info.bitness || '64';
      if (rawArch === 'arm') {
        if (os === 'android') archHint = bits === '32' ? 'armeabi-v7a' : 'arm64-v8a';
        else if (os === 'linux') archHint = bits === '32' ? 'armv7' : 'arm64';
        else archHint = 'arm64';
      } else if (rawArch === 'x86') {
        if (os === 'android') archHint = bits === '32' ? 'x86' : 'x86_64';
        else archHint = 'amd64';
      }
    } catch (e) {}
  }

  return { os, pkgIndex: matchPkgIndexByArch(os, archHint), archHint };
}

/* Exact port of SmoothCornerShape.kt (n=4 squircle with 16 steps per quadrant) */
function buildSquirclePath(w, h, radius) {
  const maxR = Math.min(w, h) / 2;
  const r = Math.max(0, Math.min(radius, maxR));
  if (r <= 0 || w <= 0 || h <= 0) {
    return `M0,0 L${w},0 L${w},${h} L0,${h} Z`;
  }
  const pts = [];
  const addCorner = (cx, cy, startDeg, endDeg, rad) => {
    const steps = 16;
    for (let step = 0; step <= steps; step++) {
      const deg = startDeg + ((endDeg - startDeg) * step) / steps;
      const radAngle = (deg * Math.PI) / 180;
      const c = Math.cos(radAngle);
      const s = Math.sin(radAngle);
      const x = cx + rad * Math.sign(c) * Math.pow(Math.abs(c), 0.5);
      const y = cy + rad * Math.sign(s) * Math.pow(Math.abs(s), 0.5);
      pts.push(`${x.toFixed(2)},${y.toFixed(2)}`);
    }
  };
  addCorner(r, r, 180, 270, r);
  addCorner(w - r, r, 270, 360, r);
  addCorner(w - r, h - r, 0, 90, r);
  addCorner(r, h - r, 90, 180, r);
  return 'M' + pts[0] + ' L' + pts.slice(1).join(' L') + ' Z';
}

function applyAllSquircles() {
  document.querySelectorAll('[data-squircle]').forEach(el => {
    const r = parseFloat(el.getAttribute('data-squircle')) || 16;
    el.style.borderRadius = r + 'px';
    const rect = el.getBoundingClientRect();
    if (rect.width > 4 && rect.height > 4) {
      const d = buildSquirclePath(rect.width, rect.height, r);
      el.style.clipPath = `path('${d}')`;
    }
  });
}

function switchView(viewName, updateHash = true) {
  state.activeView = viewName;
  ['home', 'mobile', 'desktop', 'cli'].forEach(v => {
    const el = document.getElementById('view-' + v);
    if (el) el.classList.toggle('hidden', v !== viewName);
  });
  document.querySelectorAll('.nav-tab').forEach(btn => {
    const isCurrent = btn.getAttribute('data-nav') === viewName;
    btn.classList.toggle('nav-active', isCurrent);
    btn.classList.toggle('text-monet-onSurfaceVariant', !isCurrent);
  });
  if (updateHash && location.hash !== '#' + viewName) {
    history.pushState(null, '', '#' + viewName);
  }
  window.scrollTo({ top: 0, behavior: 'smooth' });
  requestAnimationFrame(applyAllSquircles);
}

function fitCtaRow() {
  const row = document.getElementById('primaryDownloadLink').parentElement;
  if (window.innerWidth < 320) { row.classList.add('cta-stacked'); return; }
  const available = row.getBoundingClientRect().width;
  const prevW = row.style.width, prevD = row.style.flexDirection;
  row.classList.remove('cta-stacked');
  row.style.flexDirection = 'row';
  row.style.width = 'min-content';
  const required = row.getBoundingClientRect().width;
  row.style.width = prevW;
  row.style.flexDirection = prevD;
  row.classList.toggle('cta-stacked', required > available + 0.5);
}

function renderHomeOS() {
  const cfg = OS_CONFIG[state.selectedOS];
  document.querySelectorAll('.os-pill').forEach(btn => {
    const active = btn.getAttribute('data-os') === state.selectedOS;
    btn.className = active
      ? 'os-pill os-active flex items-center justify-center gap-1.5 py-2 text-xs font-medium transition-all'
      : 'os-pill flex items-center justify-center gap-1.5 py-2 text-xs font-medium text-monet-onSurfaceVariant hover:text-monet-onSurface transition-all';
  });

  const pkg = cfg.packages[state.selectedPkgIndex] || cfg.packages[0];
  const menu = document.getElementById('archDropdownMenu');
  menu.innerHTML = cfg.packages.map((p, idx) => {
    const isSel = idx === state.selectedPkgIndex;
    return `
      <button type="button" data-arch-idx="${idx}" data-squircle="12" class="arch-option w-full px-3 py-2 text-left text-xs flex items-center justify-between transition-colors ${isSel ? 'bg-monet-primaryContainer/70 text-monet-primary font-semibold' : 'text-monet-onSurface hover:bg-monet-bg'}">
        <span>${p.label}</span>
        ${isSel ? '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12l5 5l10 -10"/></svg>' : ''}
      </button>
    `;
  }).join('');

  document.querySelectorAll('.arch-option').forEach(opt => {
    opt.addEventListener('click', () => {
      state.selectedPkgIndex = parseInt(opt.getAttribute('data-arch-idx'), 10) || 0;
      menu.classList.add('hidden');
      renderHomeOS();
    });
  });
  document.getElementById('primaryDownloadLink').href = RELEASE_BASE + pkg.file;
  document.getElementById('primaryDownloadLabel').textContent = pkg.btnText;
  document.getElementById('editionTypeBadge').textContent = pkg.edition;
  document.getElementById('packageFileName').textContent = pkg.file;
  document.getElementById('archDropdownLabel').textContent = pkg.arch;
  document.getElementById('osSubtleHint').textContent = cfg.hint;
  document.getElementById('goPlatformDetailText').textContent = cfg.navLabel;

  fitCtaRow();
  requestAnimationFrame(applyAllSquircles);
}

function renderAllPackagesModal() {
  const groups = [
    { title: 'Mobile · Android 移动端（4 个包）', items: OS_CONFIG.android.packages },
    { title: 'Desktop · Windows 桌面端（2 个包）', items: OS_CONFIG.windows.packages },
    { title: 'CLI · MacOS 命令行端（2 个包）', items: OS_CONFIG.macos.packages.slice(0, 2) },
    { title: 'CLI · Linux 命令行端（3 个包）', items: OS_CONFIG.linux.packages.slice(0, 3) },
    { title: 'CLI · AI 助手技能扩展（1 个包）', items: [{ label: '[CLI Skill] 供 AI 助手调用的技能扩展包', file: 'kemaolife-skill_0.6.1.zip' }] }
  ];

  const container = document.getElementById('allPackagesContainer');
  container.innerHTML = groups.map(g => `
    <div class="space-y-2">
      <div class="font-semibold text-monet-onSurface">${g.title}</div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
        ${g.items.map(item => `
          <a href="${RELEASE_BASE + item.file}" target="_blank" rel="noopener noreferrer" data-squircle="14" class="p-3 bg-monet-bg hover:bg-monet-primaryContainer/50 flex items-center justify-between transition-colors">
            <div class="min-w-0">
              <div class="font-mono font-medium text-monet-onSurface break-all">${item.file}</div>
              <div class="text-[11px] text-monet-onSurfaceVariant">${item.label}</div>
            </div>
            <span class="text-monet-primary font-medium ml-2 shrink-0 whitespace-nowrap">下载</span>
          </a>
        `).join('')}
      </div>
    </div>
  `).join('');
}

window.addEventListener('DOMContentLoaded', () => {
  state.selectedOS = detectOSFromUA();
  state.selectedPkgIndex = matchPkgIndexByArch(state.selectedOS, detectArchHeuristic(state.selectedOS));
  renderHomeOS();

  autoDetectEnvironment().then(({ os, pkgIndex }) => {
    state.selectedOS = os;
    state.selectedPkgIndex = pkgIndex;
    renderHomeOS();
  });
  renderAllPackagesModal();
  applyAllSquircles();

  window.addEventListener('resize', () => {
    requestAnimationFrame(() => { fitCtaRow(); applyAllSquircles(); });
  });
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(() => { fitCtaRow(); applyAllSquircles(); });
  }

  document.querySelectorAll('.nav-tab').forEach(btn => {
    btn.addEventListener('click', () => {
      switchView(btn.getAttribute('data-nav'));
    });
  });

  document.getElementById('brandHomeBtn').addEventListener('click', () => {
    switchView('home');
  });

  document.querySelectorAll('.os-pill').forEach(btn => {
    btn.addEventListener('click', () => {
      state.selectedOS = btn.getAttribute('data-os');
      state.selectedPkgIndex = 0;
      renderHomeOS();
    });
  });

  const archBtn = document.getElementById('archDropdownBtn');
  const archMenu = document.getElementById('archDropdownMenu');
  const positionArchMenu = () => {
    const rect = archBtn.getBoundingClientRect();
    const menuW = 240;
    const left = Math.max(12, Math.min(window.innerWidth - menuW - 12, rect.right - menuW));
    const spaceBelow = window.innerHeight - rect.bottom;
    archMenu.style.left = left + 'px';
    if (spaceBelow < 180 && rect.top > 180) {
      archMenu.style.top = 'auto';
      archMenu.style.bottom = (window.innerHeight - rect.top + 8) + 'px';
    } else {
      archMenu.style.bottom = 'auto';
      archMenu.style.top = (rect.bottom + 8) + 'px';
    }
  };
  archBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    const willOpen = archMenu.classList.contains('hidden');
    if (willOpen) {
      positionArchMenu();
      archMenu.classList.remove('hidden');
      requestAnimationFrame(applyAllSquircles);
    } else {
      archMenu.classList.add('hidden');
    }
  });
  document.addEventListener('click', (e) => {
    if (!document.getElementById('archDropdownWrap').contains(e.target) && !archMenu.contains(e.target)) {
      archMenu.classList.add('hidden');
    }
  });
  window.addEventListener('scroll', () => {
    if (!archMenu.classList.contains('hidden')) positionArchMenu();
  }, { passive: true });

  document.getElementById('goPlatformDetailBtn').addEventListener('click', () => {
    const target = OS_CONFIG[state.selectedOS].navTarget;
    switchView(target);
  });

  const packagesModal = document.getElementById('packagesModal');
  const openPkgModal = () => {
    packagesModal.classList.remove('hidden');
    packagesModal.classList.add('flex');
    requestAnimationFrame(applyAllSquircles);
  };
  document.getElementById('openAllPackagesBtnTop').addEventListener('click', openPkgModal);
  document.querySelectorAll('[data-open-packages]').forEach(b => b.addEventListener('click', openPkgModal));

  document.getElementById('closePackagesModalBtn').addEventListener('click', () => {
    packagesModal.classList.add('hidden');
    packagesModal.classList.remove('flex');
  });
  packagesModal.addEventListener('click', (e) => {
    if (e.target === packagesModal) {
      packagesModal.classList.add('hidden');
      packagesModal.classList.remove('flex');
    }
  });

  const closeOverlays = () => {
    archMenu.classList.add('hidden');
    packagesModal.classList.add('hidden');
    packagesModal.classList.remove('flex');
  };
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeOverlays();
  });

  window.addEventListener('hashchange', () => {
    const v = location.hash.slice(1);
    if (['home', 'mobile', 'desktop', 'cli'].includes(v) && v !== state.activeView) {
      switchView(v, false);
    }
  });
  const initialView = location.hash.slice(1);
  if (['mobile', 'desktop', 'cli'].includes(initialView)) {
    switchView(initialView, false);
  }
});
