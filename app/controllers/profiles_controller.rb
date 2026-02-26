class ProfilesController < ApplicationController
  allow_unauthenticated_access

  def show
    # Mock user data - in production this would find by username
    @profile_user = {
      username: params[:username],
      display_name: "John Doe",
      bio: "Designer & Developer building cool stuff on the internet. Passionate about creating beautiful user experiences.",
      avatar_url: nil
    }

    # Mock visible links
    @links = [
      { id: 1, title: "My Portfolio", url: "https://example.com/portfolio" },
      { id: 2, title: "GitHub", url: "https://github.com/username" },
      { id: 3, title: "Twitter", url: "https://twitter.com/username" }
    ]
  end
end
