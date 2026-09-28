// OS に合ったインストールの手順を最初に見せる。違っていればタブで切り替えられる。
(function () {
  const tabs = document.querySelectorAll('.tabs button');
  const show = (os) => {
    tabs.forEach((b) => b.setAttribute('aria-selected', String(b.dataset.os === os)));
    document.querySelectorAll('.panel').forEach((p) => { p.hidden = p.dataset.os !== os; });
  };
  const ua = navigator.userAgent;
  const platform = (navigator.userAgentData && navigator.userAgentData.platform) || navigator.platform || '';
  let os = 'windows';
  if (/Mac/i.test(platform) || /Mac OS X/.test(ua)) os = 'mac';
  else if (/Linux|X11/i.test(platform) && !/Android/i.test(ua)) os = 'linux';
  tabs.forEach((b) => b.addEventListener('click', () => show(b.dataset.os)));
  show(os);

  document.querySelectorAll('.command button').forEach((btn) => {
    btn.addEventListener('click', async () => {
      const text = btn.parentElement.querySelector('code').textContent;
      try {
        await navigator.clipboard.writeText(text);
        btn.textContent = 'コピーしました';
      } catch {
        btn.textContent = '選んでコピーしてください';
      }
      setTimeout(() => { btn.textContent = 'コピー'; }, 2000);
    });
  });
})();
