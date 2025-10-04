// Dynamic Navigation Contrast Detection
class NavigationContrastDetector {
    constructor() {
        this.header = document.querySelector('header');
        this.navLinks = document.querySelectorAll('nav a');
        this.observer = null;
        this.lastContrastMode = 'default';
        this.init();
    }

    init() {
        // Initial check
        this.checkContrast();
        
        // Set up intersection observer for scroll-based detection
        this.setupScrollObserver();
        
        // Set up resize observer for responsive changes
        this.setupResizeObserver();
        
        // Periodic check for dynamic content changes
        setInterval(() => this.checkContrast(), 1000);
    }

    setupScrollObserver() {
        // Create a small element at the nav position to detect background
        const detector = document.createElement('div');
        detector.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            height: 80px;
            pointer-events: none;
            z-index: 999;
        `;
        document.body.appendChild(detector);

        this.observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    this.analyzeBackground(entry.target);
                }
            });
        }, {
            root: null,
            rootMargin: '0px',
            threshold: 0.1
        });

        // Observe all major content sections
        const sections = document.querySelectorAll('main, .home-page-content, .video-container, .cabin-container, .pricing-container');
        sections.forEach(section => this.observer.observe(section));
    }

    setupResizeObserver() {
        if (window.ResizeObserver) {
            const resizeObserver = new ResizeObserver(() => {
                this.checkContrast();
            });
            resizeObserver.observe(document.body);
        }
    }

    analyzeBackground(element) {
        // Get the computed background color/image of the element
        const computedStyle = window.getComputedStyle(element);
        const backgroundColor = computedStyle.backgroundColor;
        const backgroundImage = computedStyle.backgroundImage;
        
        // Determine if background is light or dark
        const isLight = this.isLightBackground(backgroundColor, backgroundImage);
        this.setContrastMode(isLight ? 'high-contrast' : 'dark-contrast');
    }

    isLightBackground(backgroundColor, backgroundImage) {
        // Check if background image is present (usually indicates dark content)
        if (backgroundImage && backgroundImage !== 'none') {
            return false; // Assume images are dark
        }

        // Parse RGB values from background color
        const rgbMatch = backgroundColor.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
        if (rgbMatch) {
            const r = parseInt(rgbMatch[1]);
            const g = parseInt(rgbMatch[2]);
            const b = parseInt(rgbMatch[3]);
            
            // Calculate luminance
            const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
            return luminance > 0.5; // Light if luminance > 0.5
        }

        // Check for common light colors
        const lightColors = ['white', '#fff', '#ffffff', 'transparent', 'rgba(255, 255, 255', 'rgba(248, 249, 250'];
        return lightColors.some(color => backgroundColor.toLowerCase().includes(color));
    }

    checkContrast() {
        // Get the element behind the navigation
        const navRect = this.header.getBoundingClientRect();
        const centerX = navRect.left + navRect.width / 2;
        const centerY = navRect.top + navRect.height / 2;

        // Find the element at the center of the nav
        const elementBelow = document.elementFromPoint(centerX, centerY);
        if (elementBelow && elementBelow !== this.header) {
            const computedStyle = window.getComputedStyle(elementBelow);
            const backgroundColor = computedStyle.backgroundColor;
            const backgroundImage = computedStyle.backgroundImage;
            
            const isLight = this.isLightBackground(backgroundColor, backgroundImage);
            this.setContrastMode(isLight ? 'high-contrast' : 'dark-contrast');
        }
    }

    setContrastMode(mode) {
        if (mode === this.lastContrastMode) return;

        // Remove existing contrast classes
        document.body.classList.remove('nav-high-contrast', 'nav-dark-contrast');
        
        // Add new contrast class
        if (mode !== 'default') {
            document.body.classList.add(`nav-${mode}`);
        }

        this.lastContrastMode = mode;
        
        // Add smooth transition
        this.navLinks.forEach(link => {
            link.style.transition = 'color 0.3s ease, text-shadow 0.3s ease';
        });
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new NavigationContrastDetector();
});

// Re-initialize on page changes (for SPA-like behavior)
window.addEventListener('popstate', () => {
    setTimeout(() => {
        new NavigationContrastDetector();
    }, 100);
});

