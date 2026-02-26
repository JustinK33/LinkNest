Rails.application.routes.draw do
  resource :session
  resources :passwords, param: :token

  # Reveal health status on /up that returns 200 if the app boots with no exceptions, otherwise 500.
  # Can be used by load balancers and uptime monitors to verify that the app is live.
  get "up" => "rails/health#show", as: :rails_health_check

  # Defines the root path route ("/")
  root "pages#home"

  resources :users, only: [ :new, :create ]

  # Dashboard
  get "dashboard", to: "dashboard#show", as: :dashboard

  # Links CRUD
  resources :links

  # Public profiles - must be last to not override other routes
  get "/:username", to: "profiles#show", as: :profile
end
