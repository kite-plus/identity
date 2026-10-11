// Login v2 writes its "sessions" cookie without Max-Age, so browsers drop it
// on restart and readers have to sign in again even though the server-side
// session is still valid (zitadel/zitadel#12434). Give the cookie a lifetime;
// ZITADEL still enforces each session's own expiry.
// Remove this filter once the upstream issue is fixed.

const maxAge = 30 * 24 * 60 * 60;

function persist(r) {
    const header = r.headersOut['Set-Cookie'];
    if (!header) {
        return;
    }
    const cookies = Array.isArray(header) ? header : [header];
    r.headersOut['Set-Cookie'] = cookies.map((c) => {
        if (!c.startsWith('sessions=') || /;\s*(max-age|expires)=/i.test(c)) {
            return c;
        }
        return c + '; Max-Age=' + maxAge;
    });
}

export default { persist };
