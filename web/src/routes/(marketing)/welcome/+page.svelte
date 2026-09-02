<script lang="ts">
	import ShaderGradient from '$lib/components/marketing/ShaderGradient.svelte';
	import RollingWords from '$lib/components/marketing/RollingWords.svelte';
	import { reveal, parallax, stepper } from '$lib/actions/motion';
	import Check from '@lucide/svelte/icons/check';
	import Minus from '@lucide/svelte/icons/minus';
	import Globe from '@lucide/svelte/icons/globe';
	import Wallet from '@lucide/svelte/icons/wallet';
	import Wallet2 from '@lucide/svelte/icons/badge-dollar-sign';
	import WifiOff from '@lucide/svelte/icons/wifi-off';
	import ClipboardCheck from '@lucide/svelte/icons/clipboard-check';
	import Video from '@lucide/svelte/icons/video';
	import Users from '@lucide/svelte/icons/users';
	import Layout from '@lucide/svelte/icons/layout-template';
	import Library from '@lucide/svelte/icons/library';
	import ListChecks from '@lucide/svelte/icons/list-checks';
	import FileText from '@lucide/svelte/icons/file-text';
	import Table from '@lucide/svelte/icons/table';
	import CalendarCheck from '@lucide/svelte/icons/calendar-check';
	import Award from '@lucide/svelte/icons/award';
	import Bell from '@lucide/svelte/icons/bell';
	import ShieldCheck from '@lucide/svelte/icons/shield-check';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';

	// The first question is open, so the section never reads as a wall of bars.
	let open = $state(0);

	// Which of the four steps is on screen, for the rail beside them.
	let active = $state(0);

	const institutions = ['school', 'madrasah', 'college', 'academy', 'university'];

	const reasons = [
		{
			icon: Wallet,
			title: 'Money, the way it actually moves here',
			body: 'A platform that only takes cards cannot enroll most of your learners. This one starts from bKash and a bank slip, and treats the card as the exception.'
		},
		{
			icon: Globe,
			title: 'Written for your script, not translated into it',
			body: 'Arabic and Bengali were not bolted on afterwards. Direction, punctuation and numerals were decided before the first screen was built.'
		},
		{
			icon: WifiOff,
			title: 'Assumes the connection will drop',
			body: 'Every platform works on fiber. This one is built for the bus, the village and the last week of the month, and never loses what was already done.'
		},
		{
			icon: ShieldCheck,
			title: 'You are not renting your own records',
			body: 'Isolation is enforced by the database, not by code that could forget. Everything exports in full, whenever you ask, and nothing is trained on.'
		},
		{
			icon: Wallet2,
			title: 'One price, no feature gates',
			body: 'Nobody has to buy the attendance module or the certificate add-on. The plan decides how many learners, never what you are allowed to teach.'
		},
		{
			icon: Users,
			title: 'Built by people in the same time zone',
			body: 'Support in Bangla, Arabic or English, from people who know what a Dakhil result sheet is and when the school year actually starts.'
		}
	];

	const steps = [
		{
			title: 'Open the school',
			body: 'A name, an address for your site, the script you teach in. About a minute, and no card.'
		},
		{
			title: 'Build a course',
			body: 'Sections and lessons, a pasted video link, a quiz with its answer key, an assignment with a due date.'
		},
		{
			title: 'Let the learners in',
			body: 'Invite them, or let them enroll and pay by bKash or bank transfer while a person on staff approves it.'
		},
		{
			title: 'Run the term',
			body: 'Grade what comes in, take the register, watch the gradebook fill, hand out certificates at the end.'
		}
	];

	const audiences = [
		{
			title: 'For a madrasah',
			body: 'Hifz cycles and the board syllabus side by side, guardians told the same morning a child is missing, fees collected without a card.',
			line: 'Attendance · guardian alerts · Arabic-first pages'
		},
		{
			title: 'For a school or college',
			body: 'A notice board the office can update itself, a weighted gradebook that survives a full term, and results parents can actually find.',
			line: 'Gradebook · notices · a public website'
		},
		{
			title: 'For a teacher on their own',
			body: 'One course, one price, one page that sells it. No institution behind you and no monthly fee until you have fifty learners.',
			line: 'Free to start · your own site · your own media'
		}
	];

	const comparison = [
		{ line: 'Fees by mobile wallet or a bank slip somebody approves', us: 'Built in', them: 'Card only' },
		{ line: 'Arabic and Bengali rendered properly', us: 'From the start', them: 'A translation layer' },
		{ line: 'A public website with a page builder', us: 'Included', them: 'A separate product' },
		{ line: 'Attendance that reaches the guardian', us: 'Included', them: 'An add-on' },
		{ line: 'Teaching features held back by plan', us: 'None', them: 'Most of them' },
		{ line: 'Host it yourself if you want to', us: 'Open parts', them: 'No' }
	];

	const features = [
		{
			icon: Library,
			title: 'Courses and lessons',
			body: 'Sections and lessons of any kind — a reading, a video or audio link, a PDF, a live class — reordered as the term changes, published one at a time.'
		},
		{
			icon: ListChecks,
			title: 'Quizzes',
			body: 'Single and multiple answers, true or false, a word, or a written answer. The machine grades what it can and hands you the rest.'
		},
		{
			icon: FileText,
			title: 'Assignments',
			body: 'A brief, a due date, a late policy that does the arithmetic for you, and work handed in as text or as photographs of a page.'
		},
		{
			icon: ClipboardCheck,
			title: 'A grading queue',
			body: 'Everything waiting on a person in one list: the answer beside the correct one, the points available, and a comment back to the learner.'
		},
		{
			icon: Table,
			title: 'A weighted gradebook',
			body: 'Every learner against every graded item, weighed into a course percentage. Type in a box to override, empty it to go back to the marked score.'
		},
		{
			icon: CalendarCheck,
			title: 'Attendance',
			body: 'Take the register for a class in one pass. Marking somebody absent tells them and anyone listed as their guardian, the same morning.'
		},
		{
			icon: Video,
			title: 'Live classes',
			body: 'Paste the Meet or Zoom link and everyone enrolled joins from the course, in the window you set. Attach the recording afterwards.'
		},
		{
			icon: Award,
			title: 'Certificates',
			body: 'Awarded on a finished course, carrying a serial anybody can check on a public page — no account, no login, no phone call to the office.'
		},
		{
			icon: Wallet,
			title: 'Fees and enrollment',
			body: 'bKash, SSLCommerz, or a bank slip a member of staff approves. Approving enrolls the learner on the spot.'
		},
		{
			icon: Layout,
			title: 'A website and page builder',
			body: 'Your public pages built from sections you fill in, with eight templates to start from and your catalog pulled straight from the courses you teach.'
		},
		{
			icon: Bell,
			title: 'Notifications',
			body: 'Results, absences and approvals reach people by SMS, email or a webhook into whatever you already run.'
		},
		{
			icon: Users,
			title: 'Many schools, one sign-in',
			body: 'A teacher who works at three institutions signs in once. Every school keeps its own data, enforced by the database itself.'
		}
	];


	const faq = [
		{
			q: 'Do we need a card to start?',
			a: 'No. Opening a school is free, and the first fifty learners cost nothing. When you do pay, it is by bank transfer or bKash.'
		},
		{
			q: 'Can we keep our own video hosting?',
			a: 'Yes. Paste a YouTube or Vimeo link today; media is a plug, so your own transcoder or object store drops in later without touching a lesson.'
		},
		{
			q: 'Where does our data live?',
			a: 'On the Institution plan, in your country. Everything is Postgres, and every school is isolated by the database itself, not by application code.'
		},
		{
			q: 'What if we already have a website?',
			a: 'Keep it. The pages here are optional; plenty of schools use only the catalog page and link to it from what they already have.'
		}
	];
</script>

<svelte:head>
	<title>Fajr LMS</title>
	<meta
		name="description"
		content="A learning platform for schools and teachers in South Asia and the Gulf: local payments, Arabic and Bengali, and a whole teaching week in one place."
	/>
</svelte:head>

<section class="relative isolate overflow-hidden px-6 pt-40 pb-28 sm:pt-48 sm:pb-36">
	<ShaderGradient />
	<div class="relative mx-auto max-w-4xl text-center">
		<h1
			class="font-display text-4xl leading-[1.08] font-bold text-ink sm:text-6xl lg:text-7xl"
			use:reveal={{ y: 32 }}
		>
			<!-- The lines are fixed so a longer word widens line two rather than
			     rewrapping the headline. -->
			<span class="block">Run the whole</span>
			<span class="block"><RollingWords words={institutions} /> year</span>
			<span class="block">in one place</span>
		</h1>
		<p class="mx-auto mt-7 max-w-2xl text-lg text-ink-soft sm:text-xl" use:reveal={{ delay: 0.1 }}>
			Fajr LMS teaches, grades, collects the fees and keeps the guardians informed — in Arabic,
			Bengali or English, on the phones your learners already own.
		</p>
		<div class="mt-10 flex flex-wrap justify-center gap-3" use:reveal={{ delay: 0.2 }}>
			<a class="btn" href="/start">
				Get started free
				<ArrowRight size={17} aria-hidden="true" />
			</a>
			<a class="btn btn-quiet" href="/pricing">See the pricing</a>
		</div>
		<p class="mt-5 text-sm text-ink-soft" use:reveal={{ delay: 0.3 }}>
			Free for your first fifty learners. No card to begin.
		</p>
	</div>
</section>

<section id="why" class="tinted scroll-mt-24 px-6 py-24">
	<div class="mx-auto max-w-6xl">
	<div class="mb-14 max-w-2xl" use:reveal>
		<span class="eyebrow mb-3">Why Fajr</span>
		<h2 class="font-display text-3xl font-bold sm:text-4xl">
			Why schools here choose it
		</h2>
		<p class="mt-3 mb-0 text-lg text-ink-soft">
			Most platforms are built for a card-paying learner on fiber. This one is not.
		</p>
	</div>
	<div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3" use:reveal={{ stagger: 'article' }}>
		{#each reasons as reason (reason.title)}
			<article class="card transition-colors hover:border-brand-line">
				<span class="icon-tile mb-4">
					<reason.icon size={22} aria-hidden="true" />
				</span>
				<h3 class="mb-2 text-lg font-medium">{reason.title}</h3>
				<p class="mb-0 text-sm text-ink-soft">{reason.body}</p>
			</article>
		{/each}
	</div>
	</div>
</section>

<section id="how" class="scroll-mt-24 border-y border-line bg-raised px-6 py-24">
	<div class="mx-auto grid max-w-6xl items-center gap-14 lg:grid-cols-[2fr_3fr]">
		<!-- The heading and the rail stay put while the steps come past them. -->
		<div use:reveal>
			<span class="eyebrow mb-3">How it works</span>
			<h2 class="font-display text-3xl font-bold sm:text-4xl">
				From nothing to teaching, in four steps
			</h2>
			<p class="mt-3 mb-8 text-ink-soft">
				Each one is a screen, not a project. Nobody has to call us to get through them.
			</p>

			<ol class="mb-8 flex flex-col gap-3 border-s border-line ps-5">
				{#each steps as step, index (step.title)}
					<li
						class="text-sm transition-colors duration-300"
						class:text-ink={index === active}
						class:font-medium={index === active}
						class:text-ink-faint={index !== active}
						aria-current={index === active ? 'step' : undefined}
					>
						<span class="font-mono">{String(index + 1).padStart(2, '0')}</span>
						· {step.title}
					</li>
				{/each}
			</ol>

			<a class="btn btn-quiet btn-sm" href="/start">
				Get started free
				<ArrowRight size={15} aria-hidden="true" />
			</a>
		</div>

		<ol class="steps flex flex-col gap-4" use:stepper={(index) => (active = index)}>
			{#each steps as step, index (step.title)}
				<li class="card flex min-h-80 flex-col justify-center p-8 sm:p-12" data-step>
					<span
						class="font-display text-6xl font-bold text-brand-line sm:text-7xl"
						aria-hidden="true"
					>
						{String(index + 1).padStart(2, '0')}
					</span>
					<h3 class="mt-6 font-display text-2xl font-bold sm:text-3xl">{step.title}</h3>
					<p class="mt-3 mb-0 max-w-xl text-lg text-ink-soft">{step.body}</p>
				</li>
			{/each}
		</ol>
	</div>
</section>

<!-- Bento: tiles of different weight, drifting at different speeds. -->
<section id="website" class="scroll-mt-24 mx-auto max-w-6xl px-6 py-24">
	<div class="mb-14 max-w-2xl" use:reveal>
		<span class="eyebrow mb-3">Your website</span>
		<h2 class="font-display text-3xl font-bold sm:text-4xl">
			Your website, without a web developer
		</h2>
		<p class="mt-3 mb-0 text-lg text-ink-soft">
			Every school gets a public site built from sections you fill in, in the language it teaches
			in.
		</p>
	</div>

	<div class="grid gap-5 lg:grid-cols-3">
		<article
			class="card border-brand-line bg-brand-soft lg:col-span-2 lg:row-span-2"
			use:parallax={26}
		>
			<span class="icon-tile mb-4 bg-surface">
				<Layout size={24} aria-hidden="true" />
			</span>
			<h3 class="mb-2 font-display text-2xl font-bold">Eight templates, written for you</h3>
			<p class="mb-6 max-w-lg text-ink-soft">
				A school, a college, a madrasah or a university — each written twice, once for Bangladesh
				and once for the Gulf. Notice boards, admission pages, the numbers that matter, and your
				catalog pulled straight from the courses you teach.
			</p>
			<div class="mb-6 flex flex-wrap gap-2">
				{#each ['School', 'College', 'Madrasah', 'University'] as kind (kind)}
					<span class="chip">{kind}</span>
				{/each}
				<span class="chip chip-brand">Bangladesh</span>
				<span class="chip chip-brand">The Gulf</span>
			</div>

			<ul class="flex flex-col gap-2 border-t border-line pt-5 font-mono text-xs text-ink-faint">
				{#each ['a headline', 'the notice board', 'the numbers', 'your courses', 'questions parents ask', 'an invitation'] as section, i (section)}
					<li class="flex items-center gap-3">
						<span class="w-6">{String(i + 1).padStart(2, '0')}</span>
						<span class="h-px flex-1 bg-line"></span>
						<span>{section}</span>
					</li>
				{/each}
			</ul>
		</article>

		<article class="card" use:parallax={52}>
			<span class="icon-tile mb-4"><Globe size={22} aria-hidden="true" /></span>
			<h3 class="mb-2 text-lg font-medium">Dressed for where you teach</h3>
			<p class="mb-0 text-sm text-ink-soft">
				The Gulf setting reads right to left and sets Arabic a size larger. The Bengal setting sets
				Bengali and runs a little tighter.
			</p>
		</article>

		<article class="card" use:parallax={68}>
			<span class="icon-tile mb-4"><ShieldCheck size={22} aria-hidden="true" /></span>
			<h3 class="mb-2 text-lg font-medium">Nothing can smuggle script in</h3>
			<p class="mb-0 text-sm text-ink-soft">
				Every section is plain text checked on the server. A page cannot carry markup, so a page
				cannot carry an attack.
			</p>
		</article>

		{#each audiences as audience, index (audience.title)}
			<article
				class="card flex flex-col scroll-mt-24"
				id={index === 0 ? 'who' : undefined}
				use:parallax={34}
			>
				<h3 class="mb-2 text-lg font-medium">{audience.title}</h3>
				<p class="mb-5 flex-1 text-sm text-ink-soft">{audience.body}</p>
				<p class="mb-0 font-mono text-xs text-ink-faint">{audience.line}</p>
			</article>
		{/each}
	</div>
</section>

<section id="compare" class="tinted scroll-mt-24 border-y border-line px-6 py-24">
	<div class="mx-auto max-w-6xl">
		<div class="mb-12 max-w-2xl" use:reveal>
			<span class="eyebrow mb-3">Compare</span>
			<h2 class="font-display text-3xl font-bold sm:text-4xl">
				What the big platforms leave out
			</h2>
			<p class="mt-3 mb-0 text-lg text-ink-soft">
				Not a knock on them — they were built for a different learner, in a different country.
			</p>
		</div>
		<div class="card overflow-x-auto p-0" use:reveal>
			<table class="w-full border-collapse text-sm">
				<thead>
					<tr class="border-b border-line">
						<th class="px-6 py-4 text-start font-medium">&nbsp;</th>
						<th class="w-56 px-4 py-4 font-display font-bold whitespace-nowrap text-brand-text">Fajr LMS</th>
						<th class="w-56 px-4 py-4 font-medium whitespace-nowrap text-ink-soft">The usual</th>
					</tr>
				</thead>
				<tbody>
					{#each comparison as row (row.line)}
						<tr class="border-b border-line last:border-0">
							<td class="px-6 py-4">{row.line}</td>
							<td class="px-4 py-4 text-center whitespace-nowrap">
								<span class="inline-flex items-center gap-1.5 font-medium">
									<Check class="text-brand-text" size={16} aria-hidden="true" />
									{row.us}
								</span>
							</td>
							<td class="px-4 py-4 text-center whitespace-nowrap text-ink-soft">
								<span class="inline-flex items-center gap-1.5">
									<Minus class="text-ink-faint" size={16} aria-hidden="true" />
									{row.them}
								</span>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>
</section>

<section id="features" class="scroll-mt-24 mx-auto max-w-6xl px-6 py-24">
	<div class="mb-14 max-w-2xl" use:reveal>
		<span class="eyebrow mb-3">What you get</span>
		<h2 class="font-display text-3xl font-bold sm:text-4xl">Everything the teaching week needs</h2>
		<p class="mt-3 mb-0 text-lg text-ink-soft">
			All of it on every plan. What changes with the plan is how many learners you teach.
		</p>
	</div>

	<div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3" use:reveal={{ stagger: 'article' }}>
		{#each features as feature (feature.title)}
			<article class="card transition-colors hover:border-brand-line">
				<span class="icon-tile mb-4">
					<feature.icon size={22} aria-hidden="true" />
				</span>
				<h3 class="mb-2 text-lg font-medium">{feature.title}</h3>
				<p class="mb-0 text-sm text-ink-soft">{feature.body}</p>
			</article>
		{/each}
	</div>

	<div class="mt-10 flex flex-wrap items-center gap-4" use:reveal>
		<a class="btn btn-quiet" href="/pricing">See the pricing</a>
		<p class="mb-0 text-sm text-ink-soft">
			No teaching feature is held back for a higher tier.
		</p>
	</div>
</section>

<section id="faq" class="scroll-mt-24 border-t border-line px-6 py-24">
	<div class="mx-auto max-w-3xl">
		<div class="mb-12 text-center" use:reveal>
			<span class="eyebrow mb-3">Questions</span>
			<h2 class="font-display text-3xl font-bold sm:text-4xl">Questions we are asked</h2>
		</div>

		<div class="flex flex-col gap-3" use:reveal={{ stagger: 'div.card' }}>
			{#each faq as item, i (item.q)}
				<div class="card p-0">
					<h3 class="mb-0">
						<button
							class="flex w-full items-center gap-4 px-6 py-5 text-start text-lg font-medium"
							type="button"
							aria-expanded={open === i}
							aria-controls="answer-{i}"
							onclick={() => (open = open === i ? -1 : i)}
						>
							<span class="flex-1">{item.q}</span>
							<ChevronDown
								class="shrink-0 text-ink-soft transition-transform duration-300"
								style={open === i ? 'transform: rotate(180deg)' : ''}
								size={20}
								aria-hidden="true"
							/>
						</button>
					</h3>
					<!-- Rows of zero to one fraction: the height animates without
					     anybody having to measure it. -->
					<div class="answer" class:shown={open === i} id="answer-{i}" role="region">
						<div class="min-h-0 overflow-hidden">
							<p class="mb-0 px-6 pb-5 text-ink-soft">{item.a}</p>
						</div>
					</div>
				</div>
			{/each}
		</div>
	</div>
</section>

<section class="relative isolate overflow-hidden px-6 py-32">
	<ShaderGradient />
	<div class="relative mx-auto max-w-3xl text-center" use:reveal>
		<h2 class="font-display text-4xl font-bold text-balance text-ink sm:text-5xl">
			Your school could be teaching by this evening
		</h2>
		<p class="mx-auto mt-5 max-w-xl text-lg text-ink-soft">
			Opening a school takes a name and a phone number. You can invite a teacher straight after.
		</p>
		<div class="mt-9 flex flex-wrap justify-center gap-3">
			<a class="btn" href="/start">
				Get started free
				<ArrowRight size={17} aria-hidden="true" />
			</a>
			<a class="btn btn-quiet" href="/pricing">See the pricing</a>
		</div>
	</div>
</section>

<style>
	/* Once the stepper takes over, the cards share one place and only the
	   current one is visible. */
	.steps {
		min-block-size: 20rem;
	}

	.steps:global(.stacked) {
		position: relative;
		display: block;
	}

	.steps:global(.stacked) > :global(li) {
		position: absolute;
		inset-inline: 0;
		inset-block-start: 0;
	}

	.answer {
		display: grid;
		grid-template-rows: 0fr;
		transition: grid-template-rows 320ms cubic-bezier(0.22, 1, 0.36, 1);
	}

	.shown {
		grid-template-rows: 1fr;
	}

	@media (prefers-reduced-motion: reduce) {
		.answer {
			transition: none;
		}
	}
</style>
