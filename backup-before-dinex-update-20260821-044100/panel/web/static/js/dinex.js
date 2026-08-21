(function () {
    function hideLoader() {
        var loader = document.getElementById('dinex-loader');

        if (!loader) return;

        requestAnimationFrame(function () {
            loader.classList.add('hidden');

            setTimeout(function () {
                if (loader.parentNode) {
                    loader.parentNode.removeChild(loader);
                }
            }, 300);
        });
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', hideLoader, { once: true });
    } else {
        hideLoader();
    }

    window.addEventListener('pageshow', hideLoader);
})();
