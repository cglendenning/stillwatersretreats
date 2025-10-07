document.addEventListener('DOMContentLoaded', function() {
    const stripe = Stripe('pk_live_51PTAD5Dy3ZqyS04p0EQFKzPLZHIGgA0gGsViOvfIQGFO6ZgCTBKkuVeBqdrZ09h2LmK3ym27q7bvBDvfQ5yWRUJm00JbRE1Z1P');

    function getQueryParam(param) {
        const urlParams = new URLSearchParams(window.location.search);
        const value = urlParams.get(param);
        // URLSearchParams should automatically decode, but let's be explicit
        return value ? decodeURIComponent(value) : value;
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
    
    // Debug: Log all query parameters
    console.log('All query parameters:', {
        cabin, startDate, endDate, retreatStructure, massage, meditation, hike, total,
        fullURL: window.location.href,
        search: window.location.search
    });

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
    console.log('Coaching parameter:', coaching);
    let enhancementPrice = 0;
    if (coaching) {
        console.log('Adding coaching to addOns');
        addOns.push({ name: 'Coaching', value: coaching });
        const m = coaching.match(/\$([\d,]+)/);
        if (m && m[1]) {
            enhancementPrice = parseInt(m[1].replace(/,/g, '')) || 0;
        }
    }

    console.log('Add-ons array:', addOns);
    console.log('Add-ons container:', addOnsContainer);
    if (addOns.length > 0 && addOnsContainer) {
        console.log('Rendering add-ons');
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
        console.log('No add-ons, showing default message');
        addOnsContainer.innerHTML = '<p style="text-align: center; color: #666; font-style: italic;">No enhancements selected</p>';
    }

    // Build detailed breakdown for the expandable section
    console.log('=== STARTING BREAKDOWN CALCULATION ===');
    const cleaningFee = 130;
    let cabinPrice = totalInt;
    let finalEnhancementPrice = 0;
    console.log('Initial values:', { totalInt, cabinPrice, cleaningFee });
    
    // Calculate enhancement price from coaching parameter
    if (coaching) {
        const m = coaching.match(/\$([\d,]+)/);
        if (m && m[1]) {
            finalEnhancementPrice = parseInt(m[1].replace(/,/g, '')) || 0;
            cabinPrice = totalInt - finalEnhancementPrice;
        }
    }
    
    // Calculate nightly breakdown - fetch actual rates from calendar
    const start = new Date(startDate);
    const end = new Date(endDate);
    const nights = Math.ceil((end - start) / (1000 * 60 * 60 * 24));
    const baseNightlyTotal = cabinPrice - cleaningFee;
    
    console.log('Breakdown calculation:', {
        totalInt, cabinPrice, cleaningFee, nights, baseNightlyTotal, finalEnhancementPrice
    });
    
    // Fetch actual nightly rates from calendar API
    let nightlyRates = [];
    async function fetchNightlyRates() {
        try {
            console.log('Fetching calendar data for cabin:', cabin);
            const response = await fetch(`/calendar-data?prop=${cabin}`);
            const calendarData = await response.json();
            
            console.log('Calendar API response:', calendarData);
            console.log('Calendar data type:', typeof calendarData);
            console.log('Is array?', Array.isArray(calendarData));
            
            // Handle if calendarData has a nested structure
            const dates = Array.isArray(calendarData) ? calendarData : (calendarData.dates || calendarData.data || []);
            console.log('Dates array:', dates);
            
            // Extract rates for each night of the stay
            for (let i = 0; i < nights; i++) {
                const nightDate = new Date(start);
                nightDate.setDate(start.getDate() + i);
                const dateStr = nightDate.toISOString().split('T')[0]; // YYYY-MM-DD format
                
                console.log(`Looking for date: ${dateStr}`);
                const dayData = dates.find(day => day.date === dateStr);
                console.log(`Found day data:`, dayData);
                
                if (dayData && dayData.price) {
                    nightlyRates.push({ date: nightDate, price: dayData.price });
                }
            }
            
            console.log('Fetched nightly rates:', nightlyRates);
            return nightlyRates;
        } catch (error) {
            console.error('Error fetching calendar data:', error);
            return [];
        }
    }
    
    // Get nightly prices from URL
    const nightlyPricesParam = getQueryParam('nights');
    const nightlyPrices = nightlyPricesParam ? nightlyPricesParam.split(',').map(p => parseInt(p)) : [];
    
    console.log('Nightly prices from URL:', nightlyPrices);
    
    // Populate the detailed breakdown
    const breakdownDetails = document.getElementById('breakdown-details');
    if (breakdownDetails) {
        let breakdownHTML = '';
        
        // Show each night with actual rates from URL
        if (nightlyPrices.length === nights && nights > 0) {
            for (let i = 0; i < nights; i++) {
                const nightDate = new Date(start);
                nightDate.setDate(start.getDate() + i);
                const dateStr = nightDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
                breakdownHTML += `<div class="breakdown-line"><span>Night ${i + 1} (${dateStr}):</span><span>$${nightlyPrices[i].toLocaleString()}</span></div>`;
            }
        } else {
            // Fallback to average
            console.warn('Using average nightly rate as fallback');
            const pricePerNight = nights > 0 ? Math.round(baseNightlyTotal / nights) : 0;
            for (let i = 0; i < nights; i++) {
                const nightDate = new Date(start);
                nightDate.setDate(start.getDate() + i);
                const dateStr = nightDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
                breakdownHTML += `<div class="breakdown-line"><span>Night ${i + 1} (${dateStr}):</span><span>$${pricePerNight.toLocaleString()}</span></div>`;
            }
        }
        
        // Cleaning fee
        breakdownHTML += `<div class="breakdown-line"><span>Cleaning fee:</span><span>$${cleaningFee.toLocaleString()}</span></div>`;
        
        // Enhancement if any
        if (finalEnhancementPrice > 0) {
            const enhancementName = coaching ? coaching.split('(')[0].trim() : 'Enhancement';
            breakdownHTML += `<div class="breakdown-line"><span>${enhancementName}:</span><span>$${finalEnhancementPrice.toLocaleString()}</span></div>`;
        }
        
        // Grand total
        breakdownHTML += `<div class="breakdown-line"><span>Grand Total:</span><span>$${totalInt.toLocaleString()}</span></div>`;
        
        console.log('Breakdown HTML:', breakdownHTML);
        breakdownDetails.innerHTML = breakdownHTML;
    } else {
        console.error('breakdown-details element not found');
    }

    // (Removed) Stripe Elements init is not used on review page

});

