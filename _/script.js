(async function () {
    const BACKEND = 'https://captive-portal-ochre.vercel.app/api/register';
    
    // The router IP is still sent to your Vercel backend in the payload, 
    // but it is no longer displayed on the screen.
    const routerIp = '192.168.0.1'; 

    // clientMac is defined in the HTML file by MikroTik: const clientMac = '$(mac)';
    document.getElementById('dev-mac').textContent = `MAC: ${clientMac}`;

    const phoneInput = document.getElementById('phone');
    const phoneError = document.getElementById('phone-error');
    const statusBox = document.getElementById('status');

    function showStatus(msg, type = 'error') {
        statusBox.textContent = msg;
        statusBox.className =
            'w-full max-w-md mt-4 rounded-lg px-4 py-3 text-sm ' +
            (type === 'error'
                ? 'bg-red-500/15 border border-red-500/40 text-red-300'
                : 'bg-emerald-500/15 border border-emerald-500/40 text-emerald-300');
        statusBox.classList.remove('hidden');
    }
    
    function clearStatus() {
        statusBox.classList.add('hidden');
        statusBox.textContent = '';
    }

    // If MikroTik didn't replace $(mac), it will still be the literal string '$(mac)'
    if (!clientMac || clientMac === '$(MAC)' || clientMac === '00:00:00:00:00:00') {
        document.querySelectorAll('button[data-amount]').forEach(b => {
            b.disabled = true;
            b.textContent = 'Reconnect WiFi';
        });
        showStatus('Could not detect your device MAC. Please reconnect to WiFi and try again.', 'error');
    }

    function normalizePhone(raw) {
        const digits = raw.replace(/[^\d]/g, '');
        if (/^0[17]\d{8}$/.test(digits)) return '254' + digits.slice(1);
        if (/^254[17]\d{8}$/.test(digits)) return digits;
        if (/^[17]\d{8}$/.test(digits)) return '254' + digits;
        return null;
    }
    
    phoneInput.addEventListener('input', () => {
        phoneError.classList.add('hidden');
        clearStatus();
    });

    window.pay = async function (amount, btn) {
        clearStatus();
        const normalized = normalizePhone(phoneInput.value.trim());
        if (!normalized) {
            phoneError.textContent = 'Enter a valid Safaricom number (e.g. 0712345678).';
            phoneError.classList.remove('hidden');
            phoneInput.focus();
            return;
        }

        const allButtons = document.querySelectorAll('button[data-amount]');
        const originalLabel = btn.textContent;
        allButtons.forEach(b => b.disabled = true);
        btn.textContent = 'Sending…';

        try {
            const res = await fetch(BACKEND, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    phone: normalized,
                    amount,
                    mac: clientMac,
                    router_ip: routerIp
                })
            });
            let data = null;
            try { data = await res.json(); } catch (_) { }
            if (!res.ok || (data && data.error)) {
                showStatus((data && data.error) || `Request failed (HTTP ${res.status}). Please try again.`, 'error');
            } else {
                showStatus('STK Push sent! Enter your M-Pesa PIN to gain access.', 'success');
            }
        } catch (e) {
            console.log(e);
            showStatus('Connection error. Check your network and try again.', 'error');
        } finally {
            allButtons.forEach(b => b.disabled = false);
            btn.textContent = originalLabel;
        }
    };
})();