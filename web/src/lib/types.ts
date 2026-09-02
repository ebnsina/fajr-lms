export type Membership = { tenant_id: string; role: string };

export type Tenant = {
	id: string;
	slug: string;
	name: string;
	kind: 'institution' | 'creator' | 'corporate';
	default_dir: 'auto' | 'ltr' | 'rtl';
	locale: string;
	currency: string;
	site_theme?: 'plain' | 'gulf' | 'bengal';
	role: string;
};

export type User = { id: string; full_name: string };

export type Session = {
	user: User;
	memberships: Membership[];
	tenant: Tenant | null;
};

export type ApiError = { code: string; message: string; field?: string };
