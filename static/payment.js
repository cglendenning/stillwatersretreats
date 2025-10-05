document.addEventListener('DOMContentLoaded', function() {
    const stripe = Stripe('pk_live_51PTAD5Dy3ZqyS04p0EQFKzPLZHIGgA0gGsViOvfIQGFO6ZgCTBKkuVeBqdrZ09h2LmK3ym27q7bvBDvfQ5yWRUJm00JbRE1Z1P');

    function getQueryParam(param) {
        const urlParams = new URLSearchParams(window.location.search);
        return urlParams.get(param);
    }

    async function createCheckoutSession() {
        const totalPrice = getQueryParam('total');
        const imageUrl = 'https://www.stillwatersretreats.com/assets/photo/'+getQueryParam('prop')+'/'+getQueryParam('prop')+'.jpg';
        const cabin = getQueryParam('prop');
        const startDate = getQueryParam('start');
        const endDate = getQueryParam('end');
        const retreatStructure = getQueryParam('rs');
        const massage = getQueryParam('massage');
        const meditation = getQueryParam('meditation');
        const hike = getQueryParam('hike');
        const coaching = getQueryParam('coaching');

        const response = await fetch('/create-checkout-session', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                amount: parseInt(totalPrice)*100,
                image_url: imageUrl,
                cabin: cabin,
                start_date: startDate,
                end_date: endDate,
                retreat_structure: retreatStructure,
                massage: massage,
                meditation: meditation,
                hike: hike,
                coaching: coaching
            }),
        });
        const data = await response.json();
        return data;
    }

    // Handle payment button click
    document.getElementById('pay-button').addEventListener('click', async function() {
        const button = document.getElementById('pay-button');
        
        // Show loading state
        button.disabled = true;
        button.textContent = 'Creating checkout...';
        
        try {
            const { sessionId } = await createCheckoutSession();
            
            // Redirect to Stripe checkout
            stripe.redirectToCheckout({ sessionId: sessionId })
                .then(function(result) {
                    if (result.error) {
                        alert(result.error.message);
                        button.disabled = false;
                        button.textContent = 'Book It!';
                    }
                });
                
        } catch (error) {
            console.error('Error creating checkout session:', error);
            alert('Error creating checkout session. Please try again.');
            button.disabled = false;
            button.textContent = 'Book It!';
        }
    });

    const cabin = getQueryParam('prop');
    const startDate = getQueryParam('start');
    const endDate = getQueryParam('end');
    const retreatStructure = getQueryParam('rs');
    const massage = getQueryParam('massage');
    const meditation = getQueryParam('meditation');
    const hike = getQueryParam('hike');
    const total = getQueryParam('total');

    // Convert cabin code to proper name
    let cabinName = cabin;
    switch(cabin) {
        case 'bvc':
            cabinName = 'Bear View Cove';
            break;
        case 'vp':
            cabinName = 'Victoria Pines';
            break;
    }

    // Convert dates to MM/DD/YYYY format
    function convertDateFormat(dateStr) {
        const date = new Date(dateStr);
        return date.toLocaleDateString('en-US');
    }

    // Populate the beautiful summary page
    document.getElementById('cabin-name').textContent = `${cabinName} Cabin`;
    document.getElementById('retreat-structure').textContent = retreatStructure === 'my-own' ? 'Custom Retreat Structure' : `Retreat Structure: ${retreatStructure}`;
    document.getElementById('checkin-date').textContent = convertDateFormat(startDate);
    document.getElementById('checkout-date').textContent = convertDateFormat(endDate);
    const totalInt = parseInt(total) || 0;
    document.getElementById('total-amount').textContent = `$${totalInt.toLocaleString()}`;

    // Set cabin description based on cabin type
    let cabinDescription = '';
    if (cabin === 'bvc') {
        cabinDescription = 'Mountain peace and quiet awaits you in this cozy, clean and bright perfect mountain cabin in the coveted upper moonridge area.';
    } else if (cabin === 'vp') {
        cabinDescription = 'Escape to our mountain cottage in the coveted lower moonridge neighborhood with a 5 minute walk to the free Big Bear Trolley.';
    }
    document.getElementById('cabin-description').textContent = cabinDescription;

    // Populate add-ons
    const addOnsContainer = document.getElementById('add-ons-container');
    if (addOnsContainer) { addOnsContainer.innerHTML = ''; }
    const addOns = [];
    
    if (massage && massage !== 'no' && massage !== 'None') {
        addOns.push({ name: 'Massage', value: massage });
    }
    if (meditation && meditation !== 'no' && meditation !== 'None') {
        addOns.push({ name: 'Meditation', value: meditation });
    }
    if (hike && hike !== 'no' && hike !== 'None') {
        addOns.push({ name: 'Nature Hike', value: hike });
    }

    // Coaching add-on from pricing page (when coming from pricing)
    const coaching = getQueryParam('coaching');
    let enhancementPrice = 0;
    if (coaching) {
        addOns.push({ name: 'Coaching', value: coaching });
        const m = coaching.match(/\$([\d,]+)/);
        if (m && m[1]) {
            enhancementPrice = parseInt(m[1].replace(/,/g, '')) || 0;
        }
    }

    if (addOns.length > 0 && addOnsContainer) {
        addOns.forEach(addon => {
            const addonDiv = document.createElement('div');
            addonDiv.className = 'add-on-item selected';
            // Try to extract price for display, fallback to plain label
            let line = addon.value;
            const priceMatch = addon.value.match(/\$([\d,]+)/);
            if (priceMatch) {
                line = `${addon.name}: $${priceMatch[1]}`;
            }
            addonDiv.innerHTML = `<h4>${addon.name}</h4><p>${line}</p>`;
            addOnsContainer.appendChild(addonDiv);
        });
    } else if (addOnsContainer) {
        addOnsContainer.innerHTML = '<p style="text-align: center; color: #666; font-style: italic;">No enhancements selected</p>';
    }

    // Enhanced price breakdown with clear financial breakdown
    const breakdown = [];
    let cabinPrice = totalInt;
    let enhancementPrice = 0;
    
    // Calculate enhancement price from coaching parameter
    if (coaching) {
        const m = coaching.match(/\$([\d,]+)/);
        if (m && m[1]) {
            enhancementPrice = parseInt(m[1].replace(/,/g, '')) || 0;
            cabinPrice = totalInt - enhancementPrice;
        }
    }
    
    // Build breakdown display
    if (enhancementPrice > 0) {
        breakdown.push(`Cabin: $${cabinPrice.toLocaleString()}`);
        breakdown.push(`Enhancement: $${enhancementPrice.toLocaleString()}`);
        breakdown.push(`Total: $${totalInt.toLocaleString()}`);
    } else if (addOns.length > 0) {
        breakdown.push(`Cabin: $${totalInt.toLocaleString()}`);
        breakdown.push(`Enhancements: ${addOns.map(a => a.name).join(', ')}`);
    } else {
        breakdown.push(`Cabin: $${totalInt.toLocaleString()}`);
    }
    
    const bd = document.getElementById('price-breakdown');
    if (bd) {
        bd.innerHTML = breakdown.join('<br>');
    }

    // (Removed) Stripe Elements init is not used on review page

});

