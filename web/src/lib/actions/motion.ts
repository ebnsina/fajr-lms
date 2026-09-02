import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/dist/ScrollTrigger';

let registered = false;

function ready(): boolean {
	if (typeof window === 'undefined') return false;
	if (!registered) {
		gsap.registerPlugin(ScrollTrigger);
		registered = true;
	}
	// A hidden tab has no animation frames, so an intro would leave the page
	// parked at opacity zero until somebody looked at it. Show it instead.
	if (document.hidden) return false;
	return !window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

type RevealOptions = { delay?: number; y?: number; stagger?: string };

/** Lifts an element into place as it comes into view. Elements start visible,
    so a page that never scrolls, or a reader who asked for less motion, still
    reads exactly the same. */
export function reveal(node: HTMLElement, options: RevealOptions = {}) {
	if (!ready()) return {};

	const targets = options.stagger ? Array.from(node.querySelectorAll(options.stagger)) : [node];
	if (targets.length === 0) return {};

	// Anything already on screen plays at once. Only what is below the fold waits
	// for a scroll, so the first paint is never a page of blank space.
	const onScreen = node.getBoundingClientRect().top < window.innerHeight;
	const tween = gsap.from(targets, {
		y: options.y ?? 24,
		opacity: 0,
		duration: 0.7,
		delay: options.delay ?? 0,
		ease: 'power3.out',
		stagger: options.stagger ? 0.08 : 0,
		scrollTrigger: onScreen ? undefined : { trigger: node, start: 'top 85%', once: true }
	});

	return {
		destroy() {
			tween.scrollTrigger?.kill();
			tween.kill();
		}
	};
}

/** Drifts an element against the scroll. The distance is small on purpose: a
    bento tile that wanders too far stops looking like a grid. */
export function parallax(node: HTMLElement, distance = 40) {
	if (!ready()) return {};

	const tween = gsap.to(node, {
		y: -distance,
		ease: 'none',
		scrollTrigger: { trigger: node, start: 'top bottom', end: 'bottom top', scrub: 0.6 }
	});

	return {
		destroy() {
			tween.scrollTrigger?.kill();
			tween.kill();
		}
	};
}

/** Counts a number up once it is on screen, for the figures in the hero band. */
export function countUp(node: HTMLElement, to: number) {
	if (!ready()) {
		node.textContent = String(to);
		return {};
	}

	const state = { value: 0 };
	const onScreen = node.getBoundingClientRect().top < window.innerHeight;
	const tween = gsap.to(state, {
		value: to,
		duration: 1.2,
		ease: 'power2.out',
		onUpdate: () => (node.textContent = String(Math.round(state.value))),
		scrollTrigger: onScreen ? undefined : { trigger: node, start: 'top 90%', once: true }
	});

	return {
		destroy() {
			tween.scrollTrigger?.kill();
			tween.kill();
		}
	};
}
