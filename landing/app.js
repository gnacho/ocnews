/* ocnews landing app logic */

(function () {
  'use strict';

  // Theme
  const storedTheme = localStorage.getItem('ocnews-theme');
  const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  let theme = storedTheme || (systemDark ? 'dark' : 'light');

  function applyTheme() {
    document.documentElement.setAttribute('data-theme', theme);
  }
  applyTheme();

  function updateToggleStates() {
    if (themeBtn) themeBtn.setAttribute('aria-pressed', theme === 'dark' ? 'true' : 'false');
    if (langBtn) langBtn.setAttribute('aria-pressed', currentLang === 'en' ? 'true' : 'false');
  }

  const themeBtn = document.getElementById('themeBtn');
  if (themeBtn) {
    themeBtn.addEventListener('click', () => {
      theme = theme === 'dark' ? 'light' : 'dark';
      localStorage.setItem('ocnews-theme', theme);
      applyTheme();
      updateToggleStates();
    });
  }

  // Language
  const langBtn = document.getElementById('langBtn');
  if (langBtn) {
    langBtn.addEventListener('click', () => {
      const next = currentLang === 'es' ? 'en' : 'es';
      setLang(next);
      renderSlider();
      renderGridShots();
      updateToggleStates();
    });
  }

  const urlParams = new URLSearchParams(window.location.search);
  if (urlParams.get('hl') === 'en') setLang('en');
  if (urlParams.get('hl') === 'es') setLang('es');
  applyLang();

  // Slider
  const slides = [
    { key: 'reader', file: 'shot-reader', alt: 'Reader view' },
    { key: 'article', file: 'shot-article', alt: 'Open article' }
  ];

  let currentSlide = 0;
  const shotImg = document.getElementById('shotImg');
  const thumbsEl = document.querySelector('.slider-thumbs');
  const prevBtn = document.querySelector('.slider-prev');
  const nextBtn = document.querySelector('.slider-next');

  function shotSrc(slide) {
    const themeSuffix = theme === 'dark' ? 'dark' : 'light';
    return `assets/${slide.file}-${currentLang}-${themeSuffix}.webp`;
  }

  function renderGridShots() {
    document.querySelectorAll('.shots-grid figure img').forEach((img, i) => {
      if (slides[i]) img.src = shotSrc(slides[i]);
    });
  }

  function renderSlider() {
    const slide = slides[currentSlide];
    if (shotImg) {
      shotImg.style.opacity = '0';
      setTimeout(() => {
        shotImg.src = shotSrc(slide);
        shotImg.alt = t('slider.label', { n: currentSlide + 1, total: slides.length, title: t(`shots.caption${currentSlide + 1}`) });
        shotImg.onload = () => { shotImg.style.opacity = '1'; };
        shotImg.onerror = () => { shotImg.style.opacity = '1'; };
      }, 150);
    }
    if (thumbsEl) {
      thumbsEl.innerHTML = '';
      slides.forEach((s, i) => {
        const btn = document.createElement('button');
        btn.className = 'slider-thumb' + (i === currentSlide ? ' active' : '');
        btn.setAttribute('role', 'tab');
        btn.setAttribute('aria-label', t('slider.label', { n: i + 1, total: slides.length, title: t(`shots.caption${i + 1}`) }));
        btn.setAttribute('aria-selected', i === currentSlide ? 'true' : 'false');
        const img = document.createElement('img');
        img.src = shotSrc(s);
        img.alt = '';
        btn.appendChild(img);
        btn.addEventListener('click', () => {
          currentSlide = i;
          renderSlider();
        });
        thumbsEl.appendChild(btn);
      });
    }
  }

  if (prevBtn) {
    prevBtn.addEventListener('click', () => {
      currentSlide = (currentSlide - 1 + slides.length) % slides.length;
      renderSlider();
    });
  }

  if (nextBtn) {
    nextBtn.addEventListener('click', () => {
      currentSlide = (currentSlide + 1) % slides.length;
      renderSlider();
    });
  }

  // Re-render slider and grid shots on theme change
  if (themeBtn) {
    themeBtn.addEventListener('click', () => {
      setTimeout(() => { renderSlider(); renderGridShots(); }, 10);
    });
  }

  renderSlider();
  renderGridShots();
  updateToggleStates();

  // Lightbox
  const lightbox = document.getElementById('lightbox');
  const lightboxImg = document.getElementById('lightboxImg');
  const lightboxClose = document.querySelector('.lightbox-close');

  function openLightbox(src, alt) {
    if (!lightbox || !lightboxImg) return;
    lightboxImg.src = src;
    lightboxImg.alt = alt || t('lightbox.alt');
    lightbox.hidden = false;
    document.body.style.overflow = 'hidden';
  }

  function closeLightbox() {
    if (!lightbox) return;
    lightbox.hidden = true;
    document.body.style.overflow = '';
  }

  if (shotImg) {
    shotImg.style.cursor = 'zoom-in';
    shotImg.addEventListener('click', () => {
      openLightbox(shotImg.src, shotImg.alt);
    });
  }

  document.querySelectorAll('.shots-grid img').forEach(img => {
    img.addEventListener('click', () => {
      openLightbox(img.src, img.alt);
    });
  });

  if (lightbox) {
    lightbox.addEventListener('click', (e) => {
      if (e.target === lightbox) closeLightbox();
    });
  }

  if (lightboxClose) {
    lightboxClose.addEventListener('click', closeLightbox);
  }

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeLightbox();
  });

  // Copy install commands
  const copyBtn = document.getElementById('copyInstall');
  const installCode = document.getElementById('installCode');
  if (copyBtn && installCode) {
    copyBtn.addEventListener('click', async () => {
      const text = installCode.textContent;
      try {
        await navigator.clipboard.writeText(text);
      } catch (err) {
        const ta = document.createElement('textarea');
        ta.value = text;
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
      }
      const original = copyBtn.innerHTML;
      copyBtn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"/></svg><span>' + t('install.copied') + '</span>';
      setTimeout(() => { copyBtn.innerHTML = original; applyLang(); }, 2000);
    });
  }

  // Scroll reveals
  const revealEls = document.querySelectorAll('.feature-card, .section-header, .club-inner, .honest-note, .install-card, .requirements, .about-inner, .support-card, .shots-grid figure');

  if ('IntersectionObserver' in window) {
    const io = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          entry.target.classList.add('revealed');
          io.unobserve(entry.target);
        }
      });
    }, { threshold: 0.1, rootMargin: '0px 0px -50px 0px' });

    revealEls.forEach((el, i) => {
      el.style.opacity = '0';
      el.style.transform = 'translateY(18px)';
      el.style.transition = `opacity 350ms var(--ease-out) ${Math.min(i % 6, 5) * 60}ms, transform 350ms var(--ease-out) ${Math.min(i % 6, 5) * 60}ms`;
      io.observe(el);
    });
  } else {
    revealEls.forEach(el => el.classList.add('revealed'));
  }

  const revealStyle = document.createElement('style');
  revealStyle.textContent = '.revealed { opacity: 1 !important; transform: translateY(0) !important; }';
  document.head.appendChild(revealStyle);

  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    revealEls.forEach(el => {
      el.style.opacity = '1';
      el.style.transform = 'none';
    });
  }

  // Mobile nav
  const mobileToggle = document.querySelector('.nav-mobile-toggle');
  const navLinks = document.querySelector('.nav-links');
  if (mobileToggle && navLinks) {
    mobileToggle.addEventListener('click', () => {
      const open = mobileToggle.getAttribute('aria-expanded') === 'true';
      mobileToggle.setAttribute('aria-expanded', String(!open));
      navLinks.classList.toggle('open', !open);
    });
    navLinks.querySelectorAll('a').forEach(link => {
      link.addEventListener('click', () => {
        mobileToggle.setAttribute('aria-expanded', 'false');
        navLinks.classList.remove('open');
      });
    });
  }
})();
