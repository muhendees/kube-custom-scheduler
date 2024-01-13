# Custom Kubernetes Scheduler

```
         ┌────────────────────┐
         │   Action:          │
         │   INSERT or UPDATE │
         └───────┬────────────┘
                 │
                 │
                 │
         ┌───────┼───────────────────────────────────────┐        ┌────────────────────────┐
         │       │           Database                    │        │  Custom Scheduler      │
         │   ┌───▼────────┐         ┌────────────────┐   │        │                        │
         │   │  Table:    │         │  Function:     ├───┼────────┼─► pqListener()         │
         │   │  jobs      ├─────────►  jobs_notifier │   │        │       │                │
         │   └───▲────────┘         └────────────────┘   │        │       │                │
         │       │                                       │        │       │                │
         │       │                                       │        │       │                │
         └───────┼───────────────────────────────────────┘        │       │                │
                 │                                                │       │                │
                 │                                                │       │                │
                 │       SELECT job WHERE XYZ                     │       ▼                │
                 └────────────────────────────────────────────────┼── getJob()             │
                                                                  │                        │
                                                                  └────────────────────────┘
```

Scheduler has 2 simple tasks:
- Check for new events in `jobs` table and provision new Kubernetes job depending on job type.
- Scans given namespace for jobs and associated resources to be removed older than 8 hours.

I have decided to use [Viper](https://github.com/spf13/viper) for configuration management  
and [Zerolog](https://github.com/rs/zerolog) for logging. But you can incorporate anything you want.
Basically I am trying to keep all the configuration clean by using **ENV** vars

## Setup and Preliminary work

### Kubernetes access: roles and permissions

TBD

### Container

TBD

### QUEUE

TBD

### Database

Simply you need a PostgreSQL database and connection to it. I am currently using pgsql version 14.  
Better practice would be  to make a dedicated user and give permissions to use particular table/data.  
In order to keep database connections clean and under limit we listen to the events from PostgreSQL.  
The following commands were executed using `psql` to create `FUNCTION` that notifies channel.  
This function does not work by itself so we need a triggers. Currently we depend on `jobs` table  
in both cases trigger sends a channel simple message to wake listener up. The rest is just a loop.  
So, at this step assuming you have your database up and running we'll create a function for table names `jobs`:

```SQL
CREATE OR REPLACE FUNCTION jobs_notifier() RETURNS TRIGGER
AS
$$
BEGIN
    RAISE NOTICE '%', 'Notification';
        EXECUTE FORMAT('NOTIFY job_scheduler', NULL);
    RETURN NEW;
END
$$ LANGUAGE 'plpgsql';

```

And here actual trigger that sends a message to channel

```SQL
CREATE TRIGGER jobs BEFORE INSERT OR UPDATE
       ON job
       FOR EACH ROW EXECUTE PROCEDURE jobs_notifier();

```

