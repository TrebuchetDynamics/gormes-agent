// docs.gormes.ai interactive behavior. Vanilla, no deps.
(function () {
  'use strict';

  function onReady(fn) {
    if (document.readyState !== 'loading') fn();
    else document.addEventListener('DOMContentLoaded', fn);
  }

  function setDrawer(state) {
    var sidebar = document.getElementById('docs-sidebar');
    var btn = document.querySelector('[data-testid="drawer-open"]');
    if (!sidebar) return;
    sidebar.setAttribute('data-state', state);
    if (btn) btn.setAttribute('aria-expanded', state === 'open' ? 'true' : 'false');
  }

  function initDrawer() {
    var openBtn = document.querySelector('[data-testid="drawer-open"]');
    var closeBtn = document.querySelector('[data-testid="drawer-close"]');
    var backdrop = document.querySelector('.drawer-backdrop');
    var sidebar = document.getElementById('docs-sidebar');
    if (!openBtn || !sidebar) return;

    openBtn.addEventListener('click', function () {
      var isOpen = sidebar.getAttribute('data-state') === 'open';
      setDrawer(isOpen ? 'closed' : 'open');
    });

    if (closeBtn) closeBtn.addEventListener('click', function () { setDrawer('closed'); });
    if (backdrop) backdrop.addEventListener('click', function () { setDrawer('closed'); });

    // Esc closes
    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && sidebar.getAttribute('data-state') === 'open') {
        setDrawer('closed');
      }
    });

    // Link tap inside drawer → close. Use event delegation so collapsible
    // groups added in Task 4 still get captured.
    sidebar.addEventListener('click', function (e) {
      var a = e.target.closest('a[href]');
      if (!a) return;
      if (sidebar.getAttribute('data-state') === 'open') {
        setDrawer('closed');
      }
    });
  }

  onReady(function () {
    initDrawer();
  });
})();
