// Gives new readers a preferred language. The login UI creates users
// without one, ZITADEL stores "und", and its mails then fall back to
// English. ZITADEL calls addHumanUser (an Actions v2 request target) before
// creating a user; the browser's language is not part of that call, so
// remember() notes it while the registration form passes through the proxy,
// keyed by the email address in the form.

const supported = ['zh', 'en'];
const fallback = 'zh';

function browserLanguage(r) {
    const candidates = [];
    const cookie = r.headersIn['Cookie'] || '';
    const m = cookie.match(/(?:^|;\s*)NEXT_LOCALE=([^;]+)/);
    if (m) {
        candidates.push(m[1]);
    }
    (r.headersIn['Accept-Language'] || '').split(',').forEach((part) => {
        candidates.push(part.split(';')[0].trim());
    });
    for (let i = 0; i < candidates.length; i++) {
        const lang = candidates[i].toLowerCase().split('-')[0];
        if (supported.indexOf(lang) >= 0) {
            return lang;
        }
    }
    return fallback;
}

// Content handler for the registration pages: record email -> language,
// then hand the request on to the login UI unchanged.
function remember(r) {
    if (r.method === 'POST') {
        const lang = browserLanguage(r);
        let text = r.requestText || '';
        try {
            text = decodeURIComponent(text.replace(/\+/g, ' '));
        } catch (e) {
            // Not URL-encoded; search the raw body.
        }
        const emails = text.match(/[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g) || [];
        if (r.args.email) {
            emails.push(r.args.email);
        }
        emails.forEach((e) => ngx.shared.reader_language.set(e.toLowerCase(), lang));
    }
    r.internalRedirect('@login');
}

// Actions v2 "call" target: returns the request with a preferred language.
// The body carries the new user's password; nothing here logs it.
function addHumanUser(r) {
    let ctx;
    try {
        ctx = JSON.parse(r.requestText);
    } catch (e) {
        r.return(400);
        return;
    }
    const req = ctx.request || {};
    req.profile = req.profile || {};
    if (!req.profile.preferredLanguage || req.profile.preferredLanguage === 'und') {
        const email = ((req.email || {}).email || '').toLowerCase();
        req.profile.preferredLanguage = (email && ngx.shared.reader_language.get(email)) || fallback;
    }
    r.headersOut['Content-Type'] = 'application/json';
    r.return(200, JSON.stringify(req));
}

export default { remember, addHumanUser };
