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

/** Reports which step is on screen, so a rail can follow along. Nothing is
    pinned: the page keeps scrolling normally, and each step is tall enough to
    hold the screen on its own. */
export function stepper(node: HTMLElement, onStep?: (index: number) => void) {
	if (!ready()) return {};

	const cards = Array.from(node.querySelectorAll<HTMLElement>('[data-step]'));
	if (cards.length < 2) return {};

	const triggers = cards.map((card, index) =>
		ScrollTrigger.create({
			trigger: card,
			start: 'top 60%',
			end: 'bottom 40%',
			onToggle: ({ isActive }) => isActive && onStep?.(index)
		})
	);

	const tweens = cards.map((card) =>
		gsap.from(card, {
			autoAlpha: 0,
			y: 28,
			duration: 0.6,
			ease: 'power3.out',
			scrollTrigger: { trigger: card, start: 'top 85%', once: true }
		})
	);

	return {
		destroy() {
			triggers.forEach((trigger) => trigger.kill());
			tweens.forEach((tween) => {
				tween.scrollTrigger?.kill();
				tween.kill();
			});
		}
	};
}
