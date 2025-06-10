document.addEventListener('DOMContentLoaded', function() {
    const stripe = Stripe('pk_live_51PTAD5Dy3ZqyS04p0EQFKzPLZHIGgA0gGsViOvfIQGFO6ZgCTBKkuVeBqdrZ09h2LmK3ym27q7bvBDvfQ5yWRUJm00JbRE1Z1P'); // Replace with your actual Stripe publishable key

    function getQueryParam(param) {
        const urlParams = new URLSearchParams(window.location.search);
        return urlParams.get(param);
    }

    function displaySelection(id, text) {
        const element = document.getElementById(id);
        element.textContent = text;
    }

    async function createCheckoutSession(totalPrice, imageUrl) {
        const response = await fetch('/create-checkout-session', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                amount: totalPrice,
		image_url: imageUrl
            }),
        });
        const data = await response.json();
        return data;
    }

    document.getElementById('pay-button').addEventListener('click', async function() {
        const totalPrice = getQueryParam('total');
	const imageUrl = 'https://www.stillwatersretreats.com/assets/photo/'+getQueryParam('prop')+'/'+getQueryParam('prop')+'.jpg';
        const { sessionId } = await createCheckoutSession(parseInt(totalPrice)*100, imageUrl);

        stripe.redirectToCheckout({ sessionId: sessionId })
            .then(function(result) {
                if (result.error) {
                    alert(result.error.message);
                }
            });
    });

    const cabin = getQueryParam('prop');
    const startDate = getQueryParam('start');
    const endDate = getQueryParam('end');
    const retreatStructure = getQueryParam('rs');
    const massage = getQueryParam('massage');
    const meditation = getQueryParam('meditation');
    const hike = getQueryParam('hike');
    const total = getQueryParam('total');

    displaySelection('cabin-details', `Cabin: ${cabin}`);
    displaySelection('retreat-details', `Retreat Structure: ${retreatStructure}`);
    displaySelection('massage', `Massage: ${massage}`);
    displaySelection('meditation', `Meditation: ${meditation}`);
    displaySelection('hike', `Hike: ${hike}`);
    displaySelection('total-price-details', `Total Price: ${total}`);

});

