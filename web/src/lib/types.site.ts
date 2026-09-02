export type SiteBlock = {
	type: 'hero' | 'richtext' | 'features' | 'faq' | 'stats' | 'notices' | 'courses' | 'cta';
	heading?: string;
	text?: string;
	image_url?: string;
	cta_label?: string;
	cta_href?: string;
	limit?: number;
	items?: { title: string; text?: string }[];
};

export type SitePage = {
	id: string;
	slug: string;
	title: string;
	description: string;
	dir: 'auto' | 'ltr' | 'rtl';
	blocks: SiteBlock[];
	status?: 'draft' | 'published' | 'archived';
	nav_label: string;
	nav_order: number;
	updated_at: string;
	tenant_slug?: string;
	tenant_name?: string;
};

export type SiteNavItem = {
	slug: string;
	nav_label: string;
	nav_order: number;
};

export type SiteCourse = {
	id: string;
	slug: string;
	title: string;
	summary: string;
	dir: 'auto' | 'ltr' | 'rtl';
	price_minor: number;
	currency: string;
};
