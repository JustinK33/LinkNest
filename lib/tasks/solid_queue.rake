namespace :solid_queue do
  desc "Setup Solid Queue database and tables"
  task setup: :environment do
    puts "Setting up Solid Queue database..."

    begin
      # Connect to the queue database
      ActiveRecord::Base.connected_to(role: :writing, shard: :default) do
        queue_connection = SolidQueue::Job.connection

        # Check if queue tables exist
        if queue_connection.table_exists?("solid_queue_jobs")
          puts "✅ Solid Queue tables already exist"
        else
          puts "📦 Loading Solid Queue schema..."

          queue_schema_path = Rails.root.join("db", "queue_schema.rb")
          if File.exist?(queue_schema_path)
            load queue_schema_path
            puts "✅ Solid Queue schema loaded successfully"
          else
            raise "Queue schema file not found at #{queue_schema_path}"
          end
        end

        # Verify tables exist
        required_tables = [
          "solid_queue_jobs",
          "solid_queue_recurring_tasks",
          "solid_queue_processes"
        ]

        missing_tables = required_tables.reject { |table| queue_connection.table_exists?(table) }

        if missing_tables.any?
          raise "Missing Solid Queue tables: #{missing_tables.join(', ')}"
        end

        puts "✅ All Solid Queue tables verified"
      end

    rescue => e
      puts "❌ Error setting up Solid Queue: #{e.message}"
      puts "🔄 Attempting fallback migration..."

      # Fallback: try regular migration approach
      Rake::Task["db:migrate"].invoke
      puts "✅ Fallback migration completed"
    end
  end

  desc "Reset Solid Queue database (drops and recreates tables)"
  task reset: :environment do
    puts "Resetting Solid Queue database..."

    ActiveRecord::Base.connected_to(role: :writing, shard: :default) do
      queue_connection = SolidQueue::Job.connection

      # Drop existing tables
      queue_connection.tables.each do |table|
        if table.start_with?("solid_queue_")
          puts "Dropping table: #{table}"
          queue_connection.drop_table(table, if_exists: true)
        end
      end

      # Reload schema
      queue_schema_path = Rails.root.join("db", "queue_schema.rb")
      if File.exist?(queue_schema_path)
        load queue_schema_path
        puts "✅ Solid Queue schema reloaded"
      end
    end
  end

  desc "Check Solid Queue database status"
  task status: :environment do
    puts "Checking Solid Queue database status..."

    begin
      ActiveRecord::Base.connected_to(role: :writing, shard: :default) do
        queue_connection = SolidQueue::Job.connection

        tables = queue_connection.tables.select { |t| t.start_with?("solid_queue_") }

        puts "📋 Solid Queue tables found:"
        tables.each { |table| puts "  - #{table}" }

        if tables.empty?
          puts "❌ No Solid Queue tables found"
          exit 1
        else
          puts "✅ Solid Queue database is properly configured"
        end
      end
    rescue => e
      puts "❌ Cannot connect to Solid Queue database: #{e.message}"
      exit 1
    end
  end
end

# Hook into existing Rails tasks
Rake::Task["db:prepare"].enhance do
  Rake::Task["solid_queue:setup"].invoke
end
