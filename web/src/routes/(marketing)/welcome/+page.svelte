<script lang="ts">
	import ShaderGradient from '$lib/components/marketing/ShaderGradient.svelte';
	import RollingWords from '$lib/components/marketing/RollingWords.svelte';
	import { reveal, parallax, countUp } from '$lib/actions/motion';
	import Check from '@lucide/svelte/icons/check';
	import Minus from '@lucide/svelte/icons/minus';
	import Globe from '@lucide/svelte/icons/globe';
	import Wallet from '@lucide/svelte/icons/wallet';
	import WifiOff from '@lucide/svelte/icons/wifi-off';
	import ClipboardCheck from '@lucide/svelte/icons/clipboard-check';
	import Video from '@lucide/svelte/icons/video';
	import Users from '@lucide/svelte/icons/users';
	import Layout from '@lucide/svelte/icons/layout-template';
	import ShieldCheck from '@lucide/svelte/icons/shield-check';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';

	const institutions = ['school', 'madrasah', 'college', 'academy', 'university'];

	const reasons = [
		{
			icon: Wallet,
			title: 'The payments your families already use',
			body: 'bKash, SSLCommerz and plain bank transfer with a deposit slip somebody on staff approves. No card required to enroll a single learner.'
		},
		{
			icon: Globe,
			title: 'Arabic and Bengali as first-class text',
			body: 'Right to left and left to right render properly side by side, down to the punctuation and the numerals, on every screen.'
		},
		{
			icon: WifiOff,
			title: 'Built for a phone on a weak connection',
			body: 'Progress is merged, not overwritten, so a lesson watched on the bus still counts when the signal comes back.'
		},
		{
			icon: ClipboardCheck,
			title: 'The whole teaching week',
			body: 'Lessons, quizzes, assignments, grading, a gradebook that weighs it all, attendance that tells the guardians, and certificates anyone can verify.'
		},
		{
			icon: Video,
			title: 'Your live classes, your media',
			body: 'Paste a Meet or Zoom link today. Media is pluggable, so your own transcoder drops in without changing a single lesson.'
		},
		{
			icon: Users,
			title: 'One place, many schools',
			body: 'A teacher who works at three institutions signs in once. Every school keeps its own data, enforced by the database itself.'
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

	const included = [
		'A public website with a page builder',
		'Templates for schools, colleges, madrasahs and universities',
		'Courses, modules and lessons',
		'Quizzes graded automatically or by hand',
		'Assignments with file hand-in',
		'A weighted gradebook',
		'Attendance with guardian alerts',
		'Certificates with public verification',
		'Live classes with recordings',
		'Notifications by SMS, email or webhook'
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
			class="font-display text-5xl leading-[1.05] font-extrabold tracking-tight text-balance text-ink sm:text-7xl"
			use:reveal={{ y: 32 }}
		>
			Run the whole
			<RollingWords words={institutions} />
			year in one place
		</h1>
		<p class="mx-auto mt-7 max-w-2xl text-lg text-ink-soft sm:text-xl" use:reveal={{ delay: 0.1 }}>
			Fajr LMS teaches, grades, collects the fees and keeps the guardians informed — in Arabic,
			Bengali or English, on the phones your learners already own.
		</p>
		<div class="mt-10 flex flex-wrap justify-center gap-3" use:reveal={{ delay: 0.2 }}>
			<a class="btn" href="/start">
				Open a school
				<ArrowRight size={17} aria-hidden="true" />
			</a>
			<a class="btn btn-quiet" href="/pricing">See the pricing</a>
		</div>
		<p class="mt-5 text-sm text-ink-soft" use:reveal={{ delay: 0.3 }}>
			Free while you set it up. No card to begin.
		</p>
	</div>
</section>

<section class="px-6 py-16">
	<dl class="mx-auto grid max-w-5xl gap-10 text-center sm:grid-cols-4" use:reveal={{ stagger: 'div' }}>
		<div>
			<dt class="font-display text-4xl font-bold tabular-nums"><span use:countUp={3}>3</span></dt>
			<dd class="mt-2 mb-0 text-sm text-ink-soft">scripts rendered properly</dd>
		</div>
		<div>
			<dt class="font-display text-4xl font-bold tabular-nums"><span use:countUp={4}>4</span></dt>
			<dd class="mt-2 mb-0 text-sm text-ink-soft">ways to take a fee</dd>
		</div>
		<div>
			<dt class="font-display text-4xl font-bold tabular-nums"><span use:countUp={1}>1</span></dt>
			<dd class="mt-2 mb-0 text-sm text-ink-soft">minute to open a school</dd>
		</div>
		<div>
			<dt class="font-display text-4xl font-bold tabular-nums"><span use:countUp={0}>0</span></dt>
			<dd class="mt-2 mb-0 text-sm text-ink-soft">features behind a higher tier</dd>
		</div>
	</dl>
</section>

<section class="mx-auto max-w-6xl px-6 py-24">
	<div class="mb-14 max-w-2xl" use:reveal>
		<h2 class="font-display text-3xl font-bold tracking-tight sm:text-4xl">
			Why schools here choose it
		</h2>
		<p class="mt-3 mb-0 text-lg text-ink-soft">
			Most platforms are built for a card-paying learner on fiber. This one is not.
		</p>
	</div>
	<div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3" use:reveal={{ stagger: 'article' }}>
		{#each reasons as reason (reason.title)}
			<article class="card transition-colors hover:border-line-strong">
				<reason.icon class="mb-4 text-brand-text" size={24} aria-hidden="true" />
				<h3 class="mb-2 text-lg font-medium">{reason.title}</h3>
				<p class="mb-0 text-sm text-ink-soft">{reason.body}</p>
			</article>
		{/each}
	</div>
</section>

<section class="border-y border-line bg-raised px-6 py-24">
	<div class="mx-auto max-w-6xl">
		<div class="mb-14 max-w-2xl" use:reveal>
			<h2 class="font-display text-3xl font-bold tracking-tight sm:text-4xl">
				From nothing to teaching, in four steps
			</h2>
			<p class="mt-3 mb-0 text-lg text-ink-soft">
				Each one is a screen, not a project. Nobody has to call us to get through them.
			</p>
		</div>
		<ol class="grid gap-5 sm:grid-cols-2 lg:grid-cols-4" use:reveal={{ stagger: 'li' }}>
			{#each steps as step, index (step.title)}
				<li class="card">
					<span class="mb-4 block font-mono text-sm text-brand-text">
						{String(index + 1).padStart(2, '0')}
					</span>
					<h3 class="mb-2 text-lg font-medium">{step.title}</h3>
					<p class="mb-0 text-sm text-ink-soft">{step.body}</p>
				</li>
			{/each}
		</ol>
	</div>
</section>

<!-- Bento: tiles of different weight, drifting at different speeds. -->
<section class="mx-auto max-w-6xl px-6 py-24">
	<div class="mb-14 max-w-2xl" use:reveal>
		<h2 class="font-display text-3xl font-bold tracking-tight sm:text-4xl">
			Your website, without a web developer
		</h2>
		<p class="mt-3 mb-0 text-lg text-ink-soft">
			Every school gets a public site built from sections you fill in, in the language it teaches
			in.
		</p>
	</div>

	<div class="grid gap-5 lg:grid-cols-3">
		<article class="card lg:col-span-2 lg:row-span-2" use:parallax={26}>
			<Layout class="mb-4 text-brand-text" size={26} aria-hidden="true" />
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
			<Globe class="mb-4 text-brand-text" size={24} aria-hidden="true" />
			<h3 class="mb-2 text-lg font-medium">Dressed for where you teach</h3>
			<p class="mb-0 text-sm text-ink-soft">
				The Gulf setting reads right to left and sets Arabic a size larger. The Bengal setting sets
				Bengali and runs a little tighter.
			</p>
		</article>

		<article class="card" use:parallax={68}>
			<ShieldCheck class="mb-4 text-brand-text" size={24} aria-hidden="true" />
			<h3 class="mb-2 text-lg font-medium">Nothing can smuggle script in</h3>
			<p class="mb-0 text-sm text-ink-soft">
				Every section is plain text checked on the server. A page cannot carry markup, so a page
				cannot carry an attack.
			</p>
		</article>

		{#each audiences as audience (audience.title)}
			<article class="card flex flex-col" use:parallax={34}>
				<h3 class="mb-2 text-lg font-medium">{audience.title}</h3>
				<p class="mb-5 flex-1 text-sm text-ink-soft">{audience.body}</p>
				<p class="mb-0 font-mono text-xs text-ink-faint">{audience.line}</p>
			</article>
		{/each}
	</div>
</section>

<section class="border-y border-line bg-raised px-6 py-24">
	<div class="mx-auto max-w-4xl">
		<div class="mb-12 max-w-2xl" use:reveal>
			<h2 class="font-display text-3xl font-bold tracking-tight sm:text-4xl">
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
						<th class="w-40 px-4 py-4 font-display font-bold text-brand-text">Fajr LMS</th>
						<th class="w-40 px-4 py-4 font-medium text-ink-soft">The usual</th>
					</tr>
				</thead>
				<tbody>
					{#each comparison as row (row.line)}
						<tr class="border-b border-line last:border-0">
							<td class="px-6 py-4">{row.line}</td>
							<td class="px-4 py-4 text-center">
								<span class="inline-flex items-center gap-1.5 font-medium">
									<Check class="text-brand-text" size={16} aria-hidden="true" />
									{row.us}
								</span>
							</td>
							<td class="px-4 py-4 text-center text-ink-soft">
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

<section class="mx-auto max-w-6xl px-6 py-24">
	<div class="grid gap-12 lg:grid-cols-2">
		<div use:reveal>
			<h2 class="font-display text-3xl font-bold tracking-tight sm:text-4xl">
				Everything is included
			</h2>
			<p class="mt-3 mb-6 text-lg text-ink-soft">
				There is no teaching feature behind a higher tier. What changes with the plan is how many
				learners you teach.
			</p>
			<a class="btn btn-quiet" href="/pricing">See the pricing</a>
		</div>
		<ul class="grid gap-4 sm:grid-cols-2" use:reveal={{ stagger: 'li' }}>
			{#each included as item (item)}
				<li class="flex items-start gap-2.5 text-sm">
					<Check class="mt-0.5 shrink-0 text-brand-text" size={16} aria-hidden="true" />
					{item}
				</li>
			{/each}
		</ul>
	</div>
</section>

<section class="border-t border-line px-6 py-24">
	<div class="mx-auto max-w-3xl">
		<h2 class="mb-10 font-display text-3xl font-bold tracking-tight sm:text-4xl" use:reveal>
			Questions we are asked
		</h2>
		<div class="flex flex-col gap-4" use:reveal={{ stagger: 'div.card' }}>
			{#each faq as item (item.q)}
				<div class="card">
					<h3 class="mb-2 text-lg font-medium">{item.q}</h3>
					<p class="mb-0 text-ink-soft">{item.a}</p>
				</div>
			{/each}
		</div>
	</div>
</section>

<section class="relative isolate overflow-hidden px-6 py-32">
	<ShaderGradient />
	<div class="relative mx-auto max-w-3xl text-center" use:reveal>
		<h2 class="font-display text-4xl font-extrabold tracking-tight text-balance text-ink sm:text-5xl">
			Your school could be teaching by this evening
		</h2>
		<p class="mx-auto mt-5 max-w-xl text-lg text-ink-soft">
			Opening a school takes a name and a phone number. You can invite a teacher straight after.
		</p>
		<div class="mt-9 flex flex-wrap justify-center gap-3">
			<a class="btn" href="/start">
				Open a school
				<ArrowRight size={17} aria-hidden="true" />
			</a>
			<a class="btn btn-quiet" href="/pricing">See the pricing</a>
		</div>
	</div>
</section>
