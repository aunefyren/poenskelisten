function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);
    } else {
        var login_data = false;
    }

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="title">
                            Pønskelisten
                        </div>

                        <div class="text-body" style="text-align: center;">
                            Make a wish.

                            <br>
                            <br>
                            Welcome to the front page. Not much to see here currently. Use to navigation bar and head to 'Wishlists' to manage your wishlists. Head to 'Groups' to manage and view wishlists in groups.
                        </div>
                        
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Welcome to the frontpage!';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
    } else {
        showLoggedOutMenu();
    }
}