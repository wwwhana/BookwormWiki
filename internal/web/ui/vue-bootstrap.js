(function (root) {
    'use strict';

    if (root.BookwormWikiI18n) document.documentElement.lang = root.BookwormWikiI18n.detectLocale(root);

    root.process = root.process || { env: { NODE_ENV: 'production' } };

    const applyTheme = () => {
        let preference = 'system';
        try {
            const stored = root.localStorage.getItem('bookwormwiki.theme') || root.localStorage.getItem('stash.theme');
            if (['system', 'light', 'dark'].includes(stored)) preference = stored;
        } catch (_) {
            // 시스템 설정을 사용한다.
        }
        const media = root.matchMedia && root.matchMedia('(prefers-color-scheme: dark)');
        const dark = preference === 'dark' || (preference === 'system' && media && media.matches);
        document.documentElement.dataset.bookwormwikiThemePreference = preference;
        document.documentElement.dataset.bookwormwikiTheme = dark ? 'dark' : 'light';
        document.documentElement.style.colorScheme = dark ? 'dark' : 'light';
    };

    applyTheme();
    const media = root.matchMedia && root.matchMedia('(prefers-color-scheme: dark)');
    if (media && media.addEventListener) media.addEventListener('change', applyTheme);
    root.addEventListener('storage', event => {
        if (event.key === 'bookwormwiki.theme' || event.key === 'stash.theme') applyTheme();
    });
    root.bookwormwikiConsoleApplyTheme = applyTheme;
    root.bookwormwikiVueApplyTheme = applyTheme;
}(typeof globalThis !== 'undefined' ? globalThis : window));
