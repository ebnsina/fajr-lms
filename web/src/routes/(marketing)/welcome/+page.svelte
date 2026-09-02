<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import Globe from '@lucide/svelte/icons/globe';
	import Wallet from '@lucide/svelte/icons/wallet';
	import WifiOff from '@lucide/svelte/icons/wifi-off';
	import ClipboardCheck from '@lucide/svelte/icons/clipboard-check';
	import Video from '@lucide/svelte/icons/video';
	import Users from '@lucide/svelte/icons/users';

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

	const included = [
		'A public website with a page builder',
		'Courses, modules and lessons',
		'Quizzes graded automatically or by hand',
		'Assignments with file hand-in',
		'A weighted gradebook',
		'Attendance with guardian alerts',
		'Certificates with public verification',
		'Notifications by SMS, email or webhook'
	];
</script>

<svelte:head>
	<title>Fajr LMS</title>
	<meta
		name="description"
		content="A learning platform for schools and teachers in South Asia and the Gulf: local payments, Arabic and Bengali, and a whole teaching week in one place."
	/>
</svelte:head>

<section class="border-b border-line bg-raised px-6 py-20 sm:py-28">
	<div class="mx-auto max-w-3xl text-center">
		<p class="mb-3 text-sm font-medium tracking-wide uppercase text-brand-text">
			For madrasahs, schools, coaching centers and teachers
		</p>
		<h1 class="text-4xl font-semibold sm:text-5xl">Run the whole school year in one place</h1>
		<p class="mx-auto mt-4 max-w-2xl text-lg text-ink-soft">
			Fajr LMS teaches, grades, collects the fees and keeps the guardians informed — in Arabic,
			Bengali or English, on the phones your learners already own.
		</p>
		<div class="mt-8 flex flex-wrap justify-center gap-3">
			<a class="btn" href="/start">Open a school</a>
			<a class="btn btn-quiet" href="/pricing">See the pricing</a>
		</div>
		<p class="mt-4 text-sm text-ink-soft">Free while you set it up. No card to begin.</p>
	</div>
</section>

<section class="mx-auto max-w-5xl px-6 py-16">
	<h2 class="mb-2 text-2xl font-semibold">Why schools here choose it</h2>
	<p class="mb-8 max-w-2xl text-ink-soft">
		Most platforms are built for a card-paying learner on fiber. This one is not.
	</p>
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
		{#each reasons as reason (reason.title)}
			<article class="card">
				<reason.icon class="mb-3 text-brand-text" size={22} aria-hidden="true" />
				<h3 class="mb-1 font-medium">{reason.title}</h3>
				<p class="mb-0 text-sm text-ink-soft">{reason.body}</p>
			</article>
		{/each}
	</div>
</section>

<section class="border-y border-line bg-raised px-6 py-16">
	<div class="mx-auto grid max-w-5xl gap-10 lg:grid-cols-2">
		<div>
			<h2 class="mb-2 text-2xl font-semibold">Everything is included</h2>
			<p class="mb-0 text-ink-soft">
				There is no teaching feature behind a higher tier. What changes with the plan is how many
				learners you teach.
			</p>
		</div>
		<ul class="grid gap-3 sm:grid-cols-2">
			{#each included as item (item)}
				<li class="flex items-start gap-2 text-sm">
					<Check class="mt-0.5 shrink-0 text-brand-text" size={16} aria-hidden="true" />
					{item}
				</li>
			{/each}
		</ul>
	</div>
</section>

<section class="mx-auto max-w-3xl px-6 py-20 text-center">
	<h2 class="mb-2 text-2xl font-semibold">Your school could be teaching by this evening</h2>
	<p class="mb-6 text-ink-soft">
		Opening a school takes a name and a phone number. You can invite a teacher straight after.
	</p>
	<a class="btn" href="/start">Open a school</a>
</section>
