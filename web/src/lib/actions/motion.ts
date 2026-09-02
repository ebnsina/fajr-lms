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

/** Pins a section and shows its steps one at a time as the reader scrolls
    through it, reporting which one is showing. Without motion the steps stay a
    plain list, so nothing is hidden behind an animation that never runs. */
export function stepper(node: HTMLElement, onStep?: (index: number) => void) {
	if (!ready()) return {};

	const cards = Array.from(node.querySelectorAll<HTMLElement>('[data-step]'));
	if (cards.length < 2) return {};

	node.classList.add('stacked');
	gsap.set(cards, { autoAlpha: 0, y: 28 });
	gsap.set(cards[0], { autoAlpha: 1, y: 0 });

	const timeline = gsap.timeline({
		scrollTrigger: {
			trigger: node.closest('section') ?? node,
			start: 'top top',
			end: `+=${cards.length * 55}%`,
			pin: true,
			scrub: true,
			onUpdate: ({ progress }) => {
				const index = Math.min(cards.length - 1, Math.floor(progress * cards.length));
				onStep?.(index);
			}
		}
	});

	cards.slice(1).forEach((card, i) => {
		timeline
			.to(cards[i], { autoAlpha: 0, y: -28, duration: 0.4 })
			.fromTo(card, { autoAlpha: 0, y: 28 }, { autoAlpha: 1, y: 0, duration: 0.4 }, '<0.15');
	});

	return {
		destroy() {
			node.classList.remove('stacked');
			timeline.scrollTrigger?.kill();
			timeline.kill();
		}
	};
}
