Rails.application.routes.draw do
  resource :session
  resources :passwords, param: :token

  # Reveal health status on /up that returns 200 if the app boots with no exceptions, otherwise 500.
  # Can be used by load balancers and uptime monitors to verify that the app is live.
  get "up" => "rails/health#show", as: :rails_health_check

  # Defines the root path route ("/")
  root "pages#home"

  resources :users, only: [ :new, :create, :edit, :update ]

  # Dashboard
  get "dashboard", to: "dashboard#show", as: :dashboard

  # Links CRUD + Click tracking
  resources :links
  post "/links/:id/track_click", to: "links_tracking#track_click", as: :track_click_link

  # Public profiles - slug-based routing with constraint (must be last to not override other routes)
  # SlugConstraint ensures: reserved words blocked, slug must exist in database
  get "/:slug", to: "profiles#show", as: :user_profile, constraints: SlugConstraint.new
end
